package models

import (
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jinzhu/gorm"

	"gitlab.com/teserakt/c2/internal/config"
)

// setupFunc defines a database setup function,
// returning a Database instance and a tearDown function
type setupFunc func(t *testing.T) (Database, func())

func TestDBSQLite(t *testing.T) {
	setup := func(t *testing.T) (Database, func()) {
		f, err := ioutil.TempFile(os.TempDir(), "c2TestDb-")
		if err != nil {
			t.Fatalf("Cannot create temporary file: %v", err)
		}

		dbCfg := config.DBCfg{
			Type:             DBDialectSQLite,
			File:             f.Name(),
			Passphrase:       "testpass",
			SecureConnection: config.DBSecureConnectionEnabled,
			Logging:          false,
		}

		logger := log.New(os.Stdout, "", 0)

		db, err := NewDB(dbCfg, logger)
		if err != nil {
			t.Fatalf("Cannot create db: %v", err)
		}

		if err := db.Migrate(); err != nil {
			t.Fatalf("Expected no error when migrating database, got %v", err)
		}

		tearDown := func() {
			db.Close()
			f.Close()
			os.Remove(f.Name())
		}

		return db, tearDown
	}

	testDatabase(t, setup)
}

func TestDBPostgres(t *testing.T) {
	setup := func(t *testing.T) (Database, func()) {
		dbCfg := config.DBCfg{
			Type:             DBDialectPostgres,
			Passphrase:       "testpass",
			SecureConnection: config.DBSecureConnectionInsecure,
			Host:             "127.0.0.1",
			Database:         "e4",
			Username:         "e4_c2_test",
			Password:         "teserakt4",
			Schema:           "e4_c2_test_unit",
			Logging:          true,
		}

		logger := log.New(os.Stdout, "", 0)

		db, err := NewDB(dbCfg, logger)
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "no such host"), strings.Contains(err.Error(), "connection refused"):
				t.Skipf("Cannot connect to postgres server, skipping postgres db tests: %v", err)
			default:
				t.Fatalf("Error connecting to postgres server: %v", err)
			}
		}

		db.Connection().Exec("CREATE SCHEMA e4_c2_test_unit;")

		if err := db.Migrate(); err != nil {
			t.Fatalf("Expected no error when migrating database, got %v", err)
		}

		tearDown := func() {
			db.Connection().Exec("DROP SCHEMA e4_c2_test_unit CASCADE;")
			db.Close()
		}

		return db, tearDown
	}

	testDatabase(t, setup)
}

func testDatabase(t *testing.T, setup setupFunc) {

	t.Run("Insert and Get properly insert or update and retrieve", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		expectedIDKey := IDKey{
			ID:   1,
			E4ID: []byte("someID"),
			Key:  []byte("someKey"),
		}

		err := db.InsertIDKey(expectedIDKey.E4ID, expectedIDKey.Key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		idKey, err := db.GetIDKey(expectedIDKey.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(idKey, expectedIDKey) == false {
			t.Errorf("Expected idKey to be %#v, got %#v", expectedIDKey, idKey)
		}

		expectedIDKey.Key = []byte("newKey")

		err = db.InsertIDKey(expectedIDKey.E4ID, expectedIDKey.Key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		idKey, err = db.GetIDKey(expectedIDKey.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(idKey, expectedIDKey) == false {
			t.Errorf("Expected idKey to be %#v, got %#v", expectedIDKey, idKey)
		}
	})

	t.Run("GetIDKey with unknow id return record not found error", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.GetIDKey([]byte("unknow"))
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("Insert and Get properly insert or update and retrieve", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		expectedTopicKey := TopicKey{
			ID:    1,
			Key:   []byte("some-key"),
			Topic: "someTopic",
		}

		err := db.InsertTopicKey(expectedTopicKey.Topic, expectedTopicKey.Key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		topicKey, err := db.GetTopicKey(expectedTopicKey.Topic)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(topicKey, expectedTopicKey) == false {
			t.Errorf("Expected topicKey to be %#v, got %#v", expectedTopicKey, topicKey)
		}

		expectedTopicKey.Key = []byte("newKey")
		err = db.InsertTopicKey(expectedTopicKey.Topic, expectedTopicKey.Key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		topicKey, err = db.GetTopicKey(expectedTopicKey.Topic)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(topicKey, expectedTopicKey) == false {
			t.Errorf("Expected topicKey to be %#v, got %#v", expectedTopicKey, topicKey)
		}
	})

	t.Run("GetTopicKey with unknow topic return record not found error", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.GetTopicKey("unknow")
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("Delete properly delete IDKey", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		expectedIDKey := IDKey{
			ID:   1,
			E4ID: []byte("someID"),
			Key:  []byte("someKey"),
		}

		err := db.InsertIDKey(expectedIDKey.E4ID, expectedIDKey.Key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		err = db.DeleteIDKey(expectedIDKey.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		_, err = db.GetIDKey(expectedIDKey.E4ID)
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}

	})

	t.Run("Delete unknow IDKey return record not found", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		err := db.DeleteIDKey([]byte("unknow"))
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("Delete properly delete TopicKey", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		expectedTopicKey := TopicKey{
			ID:    1,
			Key:   []byte("some-key"),
			Topic: "someTopic",
		}

		err := db.InsertTopicKey(expectedTopicKey.Topic, expectedTopicKey.Key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		err = db.DeleteTopicKey(expectedTopicKey.Topic)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		_, err = db.GetTopicKey(expectedTopicKey.Topic)
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("Delete unknow topicKey returns record not found error", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		err := db.DeleteTopicKey("unknow")
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("CountIDKeys properly count IDKeys", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		ids := [][]byte{
			[]byte("a"),
			[]byte("b"),
			[]byte("c"),
			[]byte("d"),
			[]byte("e"),
		}

		for i, id := range ids {
			c, err := db.CountIDKeys()
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if c != i {
				t.Errorf("Expected count to be %d, got %d", i, c)
			}

			err = db.InsertIDKey(id, []byte("key"))
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		}

		for i, id := range ids {
			c, err := db.CountIDKeys()
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if c != len(ids)-i {
				t.Errorf("Expected count to be %d, got %d", len(ids)-i, c)
			}

			err = db.DeleteIDKey(id)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		}
	})

	t.Run("CountTopicKeys properly count topicKeys", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		topics := []string{
			"a",
			"b",
			"c",
			"d",
			"e",
		}

		for i, topic := range topics {
			c, err := db.CountTopicKeys()
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if c != i {
				t.Fatalf("Expected count to be %d, got %d", i, c)
			}

			err = db.InsertTopicKey(topic, []byte("key"))
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
		}

		for i, topic := range topics {
			c, err := db.CountTopicKeys()
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if c != len(topics)-i {
				t.Fatalf("Expected count to be %d, got %d", len(topics)-i, c)
			}

			err = db.DeleteTopicKey(topic)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
		}
	})

	t.Run("GetAllIDKeys returns all IDKeys", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		idKeys, err := db.GetAllIDKeys()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(idKeys) != 0 {
			t.Errorf("Expected %d IDKeys, got %d", 0, len(idKeys))
		}

		expectedIDKeys := []IDKey{
			IDKey{ID: 1, E4ID: []byte("a"), Key: []byte("key1")},
			IDKey{ID: 2, E4ID: []byte("b"), Key: []byte("key2")},
			IDKey{ID: 3, E4ID: []byte("c"), Key: []byte("key3")},
		}

		for _, idKey := range expectedIDKeys {
			err = db.InsertIDKey(idKey.E4ID, idKey.Key)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		}

		idKeys, err = db.GetAllIDKeys()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(idKeys, expectedIDKeys) == false {
			t.Errorf("Expected idKeys to be %#v, got %#v", expectedIDKeys, idKeys)
		}
	})

	t.Run("GetAllTopics returns all topics", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		topics, err := db.GetAllTopics()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(topics) != 0 {
			t.Errorf("Expected %d topics, got %d", 0, len(topics))
		}

		expectedTopics := []TopicKey{
			TopicKey{ID: 1, Topic: "a", Key: []byte("key1")},
			TopicKey{ID: 2, Topic: "b", Key: []byte("key2")},
			TopicKey{ID: 3, Topic: "c", Key: []byte("key3")},
		}

		for _, topicKey := range expectedTopics {
			err = db.InsertTopicKey(topicKey.Topic, topicKey.Key)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		}

		topics, err = db.GetAllTopics()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(topics, expectedTopics) == false {
			t.Errorf("Expected idKeys to be %#v, got %#v", expectedTopics, topics)
		}
	})

	t.Run("LinkIDTopic properly link an IDKey to a TopicKey", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		idKey := IDKey{ID: 1, E4ID: []byte("i-1"), Key: []byte("key")}
		if err := db.InsertIDKey(idKey.E4ID, idKey.Key); err != nil {
			t.Fatalf("Failed to insert IDKey: %v", err)
		}

		topic := TopicKey{ID: 1, Topic: "t-1", Key: []byte("key")}
		if err := db.InsertTopicKey(topic.Topic, topic.Key); err != nil {
			t.Fatalf("Failed to insert TopicKey: %v", err)
		}

		if err := db.LinkIDTopic(idKey, topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		count, err := db.CountIDsForTopic(topic.Topic)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count to be 1, got %d", count)
		}

		count, err = db.CountTopicsForID(idKey.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count to be 1, got %d", count)
		}

		topics, err := db.GetTopicsForID(idKey.E4ID, 0, 10)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(topics, []TopicKey{topic}) == false {
			t.Errorf("Expected topics to be %#v, got %#v", []TopicKey{topic}, topics)
		}

		idKeys, err := db.GetIdsforTopic(topic.Topic, 0, 10)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(idKeys, []IDKey{idKey}) == false {
			t.Errorf("Expected idKeys to be %#v, got %#v", []IDKey{idKey}, idKeys)
		}

		if err := db.UnlinkIDTopic(idKey, topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		topics, err = db.GetTopicsForID(idKey.E4ID, 0, 10)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(topics) != 0 {
			t.Errorf("Expected no topics, got %#v", topics)
		}

		idKeys, err = db.GetIdsforTopic(topic.Topic, 0, 10)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(idKeys) != 0 {
			t.Errorf("Expected no idKeys, got %#v", idKeys)
		}

		count, err = db.CountIDsForTopic(topic.Topic)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if count != 0 {
			t.Errorf("Expected count to be 0, got %d", count)
		}

		count, err = db.CountTopicsForID(idKey.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if count != 0 {
			t.Errorf("Expected count to be 0, got %d", count)
		}
	})

	t.Run("Link with unkow records return errors", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		idKey := IDKey{E4ID: []byte("a"), Key: []byte("b")}
		topicKey := TopicKey{Topic: "c", Key: []byte("d")}

		if err := db.LinkIDTopic(idKey, topicKey); err != ErrIDKeyNoPrimaryKey {
			t.Errorf("Expected error to be %v, got %v", ErrIDKeyNoPrimaryKey, err)
		}

		if err := db.InsertIDKey(idKey.E4ID, idKey.Key); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		idKey.ID = 1

		if err := db.LinkIDTopic(idKey, topicKey); err != ErrTopicKeyNoPrimaryKey {
			t.Errorf("Expected error to be %v, got %v", ErrTopicKeyNoPrimaryKey, err)
		}
	})

	t.Run("Unlink with unkow records return errors", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		idKey := IDKey{E4ID: []byte("a"), Key: []byte("b")}
		topicKey := TopicKey{Topic: "c", Key: []byte("d")}

		if err := db.UnlinkIDTopic(idKey, topicKey); err != ErrIDKeyNoPrimaryKey {
			t.Errorf("Expected error to be %v, got %v", ErrIDKeyNoPrimaryKey, err)
		}

		if err := db.InsertIDKey(idKey.E4ID, idKey.Key); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		idKey.ID = 1

		if err := db.UnlinkIDTopic(idKey, topicKey); err != ErrTopicKeyNoPrimaryKey {
			t.Errorf("Expected error to be %v, got %v", ErrTopicKeyNoPrimaryKey, err)
		}
	})

	t.Run("GetIdsforTopic with unknow topic returns a RecordNotFound error", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.GetIdsforTopic("unknow", 0, 1)
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("GetTopicsForID with unknow topic returns a RecordNotFound error", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.GetTopicsForID([]byte("unknow"), 0, 1)
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("CountIDsForTopic returns a record not found when topic doesn't exists", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.CountIDsForTopic("unknow")
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("CountTopicsForID returns a record not found when topic doesn't exists", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.CountTopicsForID([]byte("unknow"))
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("Migrate on already migrated DB doesn't fail", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		err := db.Migrate()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
