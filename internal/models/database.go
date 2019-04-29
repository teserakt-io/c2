package models

import (
	"bytes"
	"errors"
	"log"
	"strings"

	"github.com/jinzhu/gorm"

	// Load available database drivers
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	// _ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	// _ "github.com/jinzhu/gorm/dialects/mssql"

	"gitlab.com/teserakt/c2/internal/config"
)

var (
	// ErrUnsupportedDialect is returned when creating a new database with a Config having an unsupported dialect
	ErrUnsupportedDialect = errors.New("unsupported database dialect")
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

	InsertIDKey(id, protectedkey []byte) error
	InsertTopicKey(topic string, protectedKey []byte) error
	GetIDKey(id []byte) (*IDKey, error)
	GetTopicKey(topic string) (*TopicKey, error)
	DeleteIDKey(id []byte) error
	DeleteTopicKey(topic string) error
	CountIDKeys() (int, error)
	CountTopicKeys() (int, error)
	GetAllIDKeys() ([]IDKey, error)
	GetAllTopics() ([]TopicKey, error)
	LinkIDTopic(id []byte, topic string) error
	UnlinkIDTopic(id []byte, topic string) error
	CountTopicsForID(id []byte) (int, error)
	GetTopicsForID(id []byte, offset int, count int) ([]TopicKey, error)
	CountIDsForTopic(topic string) (int, error)
	GetIdsforTopic(topic string, offset int, count int) ([]IDKey, error)
}

type gormDB struct {
	db     *gorm.DB
	config config.DBCfg
	logger *log.Logger
}

var _ Database = &gormDB{}

// NewDB creates a new database
func NewDB(config config.DBCfg, logger *log.Logger) (Database, error) {
	var db *gorm.DB
	var err error

	switch config.Type {
	case DBDialectSQLite:
		cnxStr, err := config.ConnectionString()
		if err != nil {
			return nil, err
		}

		db, err = gorm.Open(config.Type.String(), cnxStr)
		if err != nil {
			return nil, err
		}
	default:
		err = ErrUnsupportedDialect
		if err != nil {
			return nil, err
		}
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
		gdb.Connection().Exec("SET search_path TO e4_c2_test;") // What is this ?
	}

	result := gdb.Connection().AutoMigrate(
		IDKey{},
		TopicKey{},
	)

	if result.Error != nil {
		return result.Error
	}

	gdb.logger.Println("Database Migration Finished.")

	return nil
}

func (gdb *gormDB) Connection() *gorm.DB {
	return gdb.db
}

func (gdb *gormDB) Close() error {
	return gdb.db.Close()
}

func (gdb *gormDB) InsertIDKey(id, protectedkey []byte) error {
	var idkey IDKey

	gdb.db.Where(&IDKey{E4ID: id}).First(&idkey)
	if gdb.db.NewRecord(idkey) {
		idkey = IDKey{E4ID: id, Key: protectedkey}

		if result := gdb.db.Create(&idkey); result.Error != nil {
			return result.Error
		}
	} else {
		idkey.Key = protectedkey

		if result := gdb.db.Model(&idkey).Updates(idkey); result.Error != nil {
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

func (gdb *gormDB) GetIDKey(id []byte) (*IDKey, error) {
	var idkey *IDKey

	result := gdb.db.Where(&IDKey{E4ID: id}).First(idkey)

	if result.Error != nil {
		return nil, result.Error
	}

	if !bytes.Equal(id, idkey.E4ID) {
		return nil, errors.New("Internal error: struct not populated but GORM indicated success")
	}

	return idkey, nil
}

func (gdb *gormDB) GetTopicKey(topic string) (*TopicKey, error) {
	var topickey *TopicKey

	result := gdb.db.Where(&TopicKey{Topic: topic}).First(topickey)
	if result.Error != nil {
		return nil, result.Error
	}

	if strings.Compare(topickey.Topic, topic) != 0 {
		return nil, errors.New("Internal error: struct not populated but GORM indicated success")
	}

	return topickey, nil
}

func (gdb *gormDB) DeleteIDKey(id []byte) error {
	var idkey IDKey

	if result := gdb.db.Where(&IDKey{E4ID: id}).First(&idkey); result.Error != nil {
		if gorm.IsRecordNotFoundError(result.Error) {
			return errors.New("ID not found, nothing to delete")
		}
		return result.Error

	}

	// safety check:
	if !bytes.Equal(idkey.E4ID, id) {
		return errors.New("Single record not populated correctly; preventing whole DB delete")
	}

	tx := gdb.db.Begin()
	if result := tx.Model(&idkey).Association("TopicKeys").Clear(); result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if result := tx.Delete(&idkey); result.Error != nil {
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
		if gorm.IsRecordNotFoundError(result.Error) {
			return errors.New("ID not found, nothing to delete")
		}
		return result.Error
	}

	if topicKey.Topic != topic {
		return errors.New("Single record not populated correctly; preventing whole DB delete")
	}

	tx := gdb.db.Begin()
	if result := gdb.db.Model(&topicKey).Association("IDKeys").Clear(); result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if result := gdb.db.Delete(&topicKey); result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func (gdb *gormDB) CountIDKeys() (int, error) {
	var idkey IDKey
	var count int
	if result := gdb.db.Model(&idkey).Count(&count); result.Error != nil {
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

func (gdb *gormDB) GetAllIDKeys() ([]IDKey, error) {
	var idkeys []IDKey
	if result := gdb.db.Find(&idkeys); result.Error != nil {
		return nil, result.Error
	}

	return idkeys, nil
}

func (gdb *gormDB) GetAllTopics() ([]TopicKey, error) {
	var topickeys []TopicKey
	if result := gdb.db.Find(&topickeys); result.Error != nil {
		return nil, result.Error
	}

	return topickeys, nil
}

/* -- M2M Functions -- */

// This function links a topic and an id/key. The link is created in both
// directions (IDkey to Topics, Topic to IDkeys).
func (gdb *gormDB) LinkIDTopic(id []byte, topic string) error {

	var idkey IDKey
	var topickey TopicKey

	if err := gdb.db.Where(&IDKey{E4ID: id}).First(&idkey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return errors.New("ID Key not found, cannot link to topic")
		}
		return err

	}
	if err := gdb.db.Where(&TopicKey{Topic: topic}).First(&topickey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return errors.New("ID Key not found, cannot link to IDkey")
		}
		return err
	}

	tx := gdb.db.Begin()
	if err := tx.Model(&idkey).Association("TopicKeys").Append(&topickey).Error; err != nil {
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
func (gdb *gormDB) UnlinkIDTopic(id []byte, topic string) error {

	var idkey IDKey
	var topickey TopicKey

	if err := gdb.db.Where(&IDKey{E4ID: id}).First(&idkey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return errors.New("ID Key not found, cannot unlink from topic")
		}
		return err
	}
	if err := gdb.db.Where(&TopicKey{Topic: topic}).First(&topickey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return errors.New("ID Key not found, cannot unlink from IDkey")
		}
		return err
	}

	tx := gdb.db.Begin()
	if err := tx.Model(&idkey).Association("TopicKeys").Delete(&topickey).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where(&IDKey{E4ID: id}).First(&idkey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			tx.Rollback()
			return errors.New("ID/Client appears to have been deleted, this is just an unlink")
		}
	}
	if err := tx.Where(&TopicKey{Topic: topic}).First(&topickey).Error; err != nil {
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

func (gdb *gormDB) GetTopicsForID(id []byte, offset int, count int) ([]TopicKey, error) {

	var idkey IDKey
	var topickeys []TopicKey

	if err := gdb.db.Where(&IDKey{E4ID: id}).First(&idkey).Error; err != nil {
		return nil, err
	}

	if err := gdb.db.Model(&idkey).Offset(offset).Limit(count).Related(&topickeys, "TopicKeys").Error; err != nil {
		return nil, err
	}

	return topickeys, nil
}

func (gdb *gormDB) CountIDsForTopic(topic string) (int, error) {
	var topickey TopicKey

	if err := gdb.db.Where(&TopicKey{Topic: topic}).First(&topickey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return 0, errors.New("Topic key not found, cannot link to ID")
		}
		return 0, err
	}

	count := gdb.db.Model(&topickey).Association("IDKeys").Count()
	return count, nil
}

func (gdb *gormDB) CountTopicsForID(id []byte) (int, error) {

	var idkey IDKey

	if err := gdb.db.Where(&IDKey{E4ID: id}).First(&idkey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return 0, errors.New("ID Key not found, cannot link to topic")
		}
		return 0, err
	}

	count := gdb.db.Model(&idkey).Association("TopicKeys").Count()

	return count, nil
}

func (gdb *gormDB) GetIdsforTopic(topic string, offset int, count int) ([]IDKey, error) {

	var topickey TopicKey
	var idkeys []IDKey

	if err := gdb.db.Where(&TopicKey{Topic: topic}).First(&topickey).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, errors.New("ID Key not found, cannot link to topic")
		}
		return nil, err
	}

	if err := gdb.db.Model(&topickey).Offset(offset).Limit(count).Related(&idkeys, "IDKeys").Error; err != nil {
		return nil, err
	}

	return idkeys, nil
}
