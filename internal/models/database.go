package models

//go:generate mockgen -destination=database_mocks.go -package models -self_package github.com/teserakt-io/c2/internal/models github.com/teserakt-io/c2/internal/models Database

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jinzhu/gorm"

	// Load available database drivers
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	// _ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	// _ "github.com/jinzhu/gorm/dialects/mssql"

	e4crypto "github.com/teserakt-io/e4go/crypto"

	"github.com/teserakt-io/c2/internal/config"
)

// QueryLimit defines the maximum number of records returned
const QueryLimit = 100

var (
	// ErrUnsupportedDialect is returned when creating a new database with a Config having an unsupported dialect
	ErrUnsupportedDialect = errors.New("unsupported database dialect")
	// ErrTopicKeyNotFound is returned when the topic cannot be found in the database
	ErrTopicKeyNotFound = errors.New("topicKey not found in database")
	// ErrClientNotFound is returned when the key cannot be found in the database
	ErrClientNotFound = errors.New("Client not found in database")
	// ErrClientNoPrimaryKey is returned when an Client is provided but it doesn't have a primary key set
	ErrClientNoPrimaryKey = errors.New("Client doesn't have primary key")
	// ErrTopicKeyNoPrimaryKey is returned when an TopicKey is provided but it doesn't have a primary key set
	ErrTopicKeyNoPrimaryKey = errors.New("TopicKey doesn't have primary key")
)

// List of available DB dialects
const (
	DBDialectSQLite   = "sqlite3"
	DBDialectPostgres = "postgres"
)

// Database describes a generic database implementation
type Database interface {
	Close() error
	Connection() *gorm.DB
	Migrate() error

	// Client Only Manipulation
	InsertClient(name string, id, protectedkey []byte) error
	GetClientByID(id []byte) (Client, error)
	DeleteClientByID(id []byte) error
	CountClients() (int, error)
	GetAllClients() ([]Client, error)
	GetClientsRange(offset, limit int) ([]Client, error)

	// Individual Topic Manipulaton
	InsertTopicKey(topic string, protectedKey []byte) error
	GetTopicKey(topic string) (TopicKey, error)
	DeleteTopicKey(topic string) error
	CountTopicKeys() (int, error)
	GetAllTopics() ([]TopicKey, error)
	GetAllTopicsUnsafe() ([]TopicKey, error)
	GetTopicsRange(offset, limit int) ([]TopicKey, error)

	// Linking, removing topic-client mappings:
	LinkClientTopic(client Client, topicKey TopicKey) error
	UnlinkClientTopic(client Client, topicKey TopicKey) error

	// > Counting topics per client, or clients per topic.
	CountTopicsForClientByID(id []byte) (int, error)
	CountClientsForTopic(topic string) (int, error)

	// > Retrieving clients per topic or topics per client
	GetTopicsForClientByID(id []byte, offset int, count int) ([]TopicKey, error)
	GetClientsForTopic(topic string, offset int, count int) ([]Client, error)
}

type gormDB struct {
	db     *gorm.DB
	config config.DBCfg
	logger *log.Logger
}

var _ Database = (*gormDB)(nil)

// NewDB creates a new database
func NewDB(config config.DBCfg, logger *log.Logger) (Database, error) {
	var db *gorm.DB
	var err error

	cnxStr, err := config.ConnectionString()
	if err != nil {
		return nil, err
	}

	db, err = gorm.Open(config.Type.String(), cnxStr)
	if err != nil {
		return nil, err
	}

	db.LogMode(config.Logging)
	db.SetLogger(logger)

	return &gormDB{
		db:     db,
		config: config,
		logger: logger,
	}, nil
}

func (gdb *gormDB) Migrate() error {
	gdb.logger.Println("Database Migration Started.")

	switch gdb.config.Type {
	case DBDialectSQLite:
		// Enable foreign key support for sqlite3
		gdb.Connection().Exec("PRAGMA foreign_keys = ON")
	case DBDialectPostgres:
		gdb.Connection().Exec(fmt.Sprintf("SET search_path TO %s;", gdb.config.Schema))
	}

	result := gdb.Connection().AutoMigrate(
		Client{},
		TopicKey{},
	)
	if result.Error != nil {
		return result.Error
	}

	switch gdb.config.Type {
	case DBDialectPostgres:
		// Postgres require to add relations manually as AutoMigrate won't do it.
		// see: https://github.com/jinzhu/gorm/issues/450

		// Add foreign key on client id
		exists, err := gdb.pgCheckConstraint("clients_topickeys_client_fk", "clients_topickeys")
		if err != nil {
			return err
		}

		if !exists {
			if err := gdb.Connection().Exec("ALTER TABLE clients_topickeys ADD CONSTRAINT clients_topickeys_client_fk FOREIGN KEY(client_id) REFERENCES clients (id) ON DELETE CASCADE;").Error; err != nil {
				return err
			}
		}

		// Add foreign key on topic id
		exists, err = gdb.pgCheckConstraint("clients_topickeys_client_fk", "clients_topickeys")
		if err != nil {
			return err
		}

		if !exists {
			if err := gdb.Connection().Exec("ALTER TABLE clients_topickeys ADD CONSTRAINT clients_topickeys_client_fk FOREIGN KEY(client_id) REFERENCES clients (id) ON DELETE CASCADE;").Error; err != nil {
				return err
			}
		}
	}

	gdb.logger.Println("Database Migration Finished.")

	return nil
}

// pgCheckConstraint probe the db to check if a foreign key with `name` exists on `table`
// This method only support Postgres dialect and will return an error otherwise.
func (gdb *gormDB) pgCheckConstraint(name, table string) (bool, error) {
	if gdb.config.Type != DBDialectPostgres {
		return false, errors.New("invalid db dialect, only Postgres is supported")
	}

	type constraintCounter struct {
		Count int
	}

	var counter constraintCounter
	result := gdb.Connection().Raw(`SELECT COUNT(1) FROM information_schema.table_constraints WHERE constraint_name=? AND table_name=?;`, name, table).Scan(&counter)

	return counter.Count > 0, result.Error
}

func (gdb *gormDB) Connection() *gorm.DB {
	return gdb.db
}

func (gdb *gormDB) Close() error {
	return gdb.db.Close()
}

func (gdb *gormDB) InsertClient(name string, id, protectedkey []byte) error {
	var client Client

	// we will actually allow empty names if necessary. If the alias
	// is unknown to the C2, then we can insert it as is and use the ID
	// based functions.
	// If the name is known, we must have H(name)==ID. Enforce this here:
	if name != "" {
		idTest := e4crypto.HashIDAlias(name)
		if !bytes.Equal(id, idTest) {
			return errors.New("H(Name) != E4ID, refusing to create or update client")
		}
	} else {
		if len(id) != e4crypto.IDLen {
			return fmt.Errorf("ID Length invalid: got %d, expected %d", len(id), e4crypto.IDLen)
		}
	}

	gdb.db.Where(&Client{E4ID: id}).First(&client)
	if gdb.db.NewRecord(client) {
		client = Client{Name: name, E4ID: id, Key: protectedkey}

		if result := gdb.db.Create(&client); result.Error != nil {
			return result.Error
		}
	} else {
		client.Key = protectedkey

		if result := gdb.db.Model(&client).Updates(client); result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func (gdb *gormDB) InsertTopicKey(topic string, protectedKey []byte) error {
	var topicKey TopicKey

	gdb.db.Where(&TopicKey{Topic: topic}).First(&topicKey)
	if gdb.db.NewRecord(topicKey) {
		topicKey = TopicKey{Topic: topic, Key: protectedKey}
		if result := gdb.db.Create(&topicKey); result.Error != nil {
			return result.Error
		}
	} else {
		topicKey.Key = protectedKey
		if result := gdb.db.Model(&topicKey).Updates(topicKey); result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func (gdb *gormDB) GetClientByID(id []byte) (Client, error) {
	var client Client

	result := gdb.db.Where(&Client{E4ID: id}).First(&client)

	if result.Error != nil {
		return Client{}, result.Error
	}

	if !bytes.Equal(id, client.E4ID) {
		return Client{}, errors.New("internal error: struct not populated but GORM indicated success")
	}

	return client, nil
}

func (gdb *gormDB) GetTopicKey(topic string) (TopicKey, error) {
	var topickey TopicKey

	result := gdb.db.Where(&TopicKey{Topic: topic}).First(&topickey)
	if result.Error != nil {
		return TopicKey{}, result.Error
	}

	if strings.Compare(topickey.Topic, topic) != 0 {
		return TopicKey{}, errors.New("internal error: struct not populated but GORM indicated success")
	}

	return topickey, nil
}

func (gdb *gormDB) DeleteClientByID(id []byte) error {
	var client Client

	if result := gdb.db.Where(&Client{E4ID: id}).First(&client); result.Error != nil {
		return result.Error
	}

	// safety check:
	if !bytes.Equal(client.E4ID, id) {
		return errors.New("single record not populated correctly; preventing whole DB delete")
	}

	tx := gdb.db.Begin()
	if result := tx.Model(&client).Association("TopicKeys").Clear(); result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if result := tx.Delete(&client); result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

func (gdb *gormDB) DeleteTopicKey(topic string) error {
	var topicKey TopicKey
	if result := gdb.db.Where(&TopicKey{Topic: topic}).First(&topicKey); result.Error != nil {
		return result.Error
	}

	if topicKey.Topic != topic {
		return errors.New("single record not populated correctly; preventing whole DB delete")
	}

	tx := gdb.db.Begin()
	if result := tx.Model(&topicKey).Association("Clients").Clear(); result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if result := tx.Delete(&topicKey); result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func (gdb *gormDB) CountClients() (int, error) {
	var client Client
	var count int
	if result := gdb.db.Model(&client).Count(&count); result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (gdb *gormDB) CountTopicKeys() (int, error) {
	var topickey TopicKey
	var count int
	if result := gdb.db.Model(&topickey).Count(&count); result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (gdb *gormDB) GetAllClients() ([]Client, error) {
	var clients []Client
	if result := gdb.db.Order("name").Limit(QueryLimit).Find(&clients); result.Error != nil {
		return nil, result.Error
	}

	return clients, nil
}

func (gdb *gormDB) GetAllTopics() ([]TopicKey, error) {
	var topickeys []TopicKey
	if result := gdb.db.Order("topic").Limit(QueryLimit).Find(&topickeys); result.Error != nil {
		return nil, result.Error
	}

	return topickeys, nil
}

func (gdb *gormDB) GetAllTopicsUnsafe() ([]TopicKey, error) {
	var topickeys []TopicKey
	if result := gdb.db.Find(&topickeys); result.Error != nil {
		return nil, result.Error
	}

	return topickeys, nil
}

func (gdb *gormDB) GetClientsRange(offset, count int) ([]Client, error) {
	var clients []Client

	if count > QueryLimit {
		count = QueryLimit
	}

	if result := gdb.db.Order("name").Offset(offset).Limit(count).Find(&clients); result.Error != nil {
		return nil, result.Error
	}

	return clients, nil
}

func (gdb *gormDB) GetTopicsRange(offset, count int) ([]TopicKey, error) {
	var topickeys []TopicKey

	if count > QueryLimit {
		count = QueryLimit
	}

	if result := gdb.db.Order("topic").Offset(offset).Limit(count).Find(&topickeys); result.Error != nil {
		return nil, result.Error
	}

	return topickeys, nil
}

/* -- M2M Functions -- */

// This function links a topic and an id/key. The link is created in both
// directions (IDkey to Topics, Topic to IDkeys).
func (gdb *gormDB) LinkClientTopic(client Client, topicKey TopicKey) error {
	if gdb.db.NewRecord(client) {
		return ErrClientNoPrimaryKey
	}

	if gdb.db.NewRecord(topicKey) {
		return ErrTopicKeyNoPrimaryKey
	}

	tx := gdb.db.Begin()
	if err := tx.Model(&client).Association("TopicKeys").Append(&topicKey).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

// This function removes the relationship between a Topic and an ID, but
// does not delete the Topic or the ID.
func (gdb *gormDB) UnlinkClientTopic(client Client, topicKey TopicKey) error {
	if gdb.db.NewRecord(client) {
		return ErrClientNoPrimaryKey
	}

	if gdb.db.NewRecord(topicKey) {
		return ErrTopicKeyNoPrimaryKey
	}

	tx := gdb.db.Begin()
	if err := tx.Model(&client).Association("TopicKeys").Delete(&topicKey).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where(&Client{ID: client.ID}).First(&client).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			tx.Rollback()
			return errors.New("ID/Client appears to have been deleted, this is just an unlink")
		}
	}
	if err := tx.Where(&TopicKey{ID: topicKey.ID}).First(&topicKey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			tx.Rollback()
			return errors.New("Topic appears to have been deleted, this is just an unlink")
		}
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func (gdb *gormDB) GetTopicsForClientByID(id []byte, offset int, count int) ([]TopicKey, error) {
	var client Client
	var topickeys []TopicKey

	if err := gdb.db.Where(&Client{E4ID: id}).First(&client).Error; err != nil {
		return nil, err
	}

	if err := gdb.db.Model(&client).Order("topic").Offset(offset).Limit(count).Related(&topickeys, "TopicKeys").Error; err != nil {
		return nil, err
	}

	return topickeys, nil
}

func (gdb *gormDB) CountClientsForTopic(topic string) (int, error) {
	var topickey TopicKey

	if err := gdb.db.Where(&TopicKey{Topic: topic}).First(&topickey).Error; err != nil {
		return 0, err
	}

	count := gdb.db.Model(&topickey).Association("Clients").Count()
	return count, nil
}

func (gdb *gormDB) CountTopicsForClientByID(id []byte) (int, error) {
	var client Client

	if err := gdb.db.Where(&Client{E4ID: id}).First(&client).Error; err != nil {
		return 0, err
	}

	count := gdb.db.Model(&client).Association("TopicKeys").Count()

	return count, nil
}

func (gdb *gormDB) GetClientsForTopic(topic string, offset int, count int) ([]Client, error) {
	var topickey TopicKey
	var clients []Client

	if err := gdb.db.Where(&TopicKey{Topic: topic}).First(&topickey).Error; err != nil {
		return nil, err
	}

	if err := gdb.db.Model(&topickey).Order("name").Offset(offset).Limit(count).Related(&clients, "Clients").Error; err != nil {
		return nil, err
	}

	return clients, nil
}

// IsErrRecordNotFound indicate whenever the err is a gorm.RecordNotFound error
func IsErrRecordNotFound(err error) bool {
	return gorm.IsRecordNotFoundError(err)
}
