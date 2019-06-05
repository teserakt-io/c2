package models

import (
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/jinzhu/gorm"

	"gitlab.com/teserakt/c2/internal/config"
	e4 "gitlab.com/teserakt/e4common"
)

// setupFunc defines a database setup function,
// returning a Database instance and a tearDown function
type setupFunc func(t *testing.T) (Database, func())

func TestDBSQLite(t *testing.T) {
	setup := func(t *testing.T) (Database, func()) {
		f, err := ioutil.TempFile(os.TempDir(), "c2TestDb")
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
	if os.Getenv("C2TEST_POSTGRES") == "" {
		t.Skip("C2TEST_POSTGRES environment variable isn't set, skipping postgress tests")
	}

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
			t.Fatalf("Error connecting to postgres server: %v", err)
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

		expectedClient := Client{
			ID:   1,
			Name: "expectedName",
			E4ID: e4.HashIDAlias("expectedName"),
			Key:  []byte("someKey"),
		}

		err := db.InsertClient(expectedClient.Name, expectedClient.E4ID, expectedClient.Key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		client, err := db.GetClientByID(expectedClient.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(client, expectedClient) == false {
			t.Errorf("Expected client to be %#v, got %#v", expectedClient, client)
		}

		expectedClient.Key = []byte("newKey")

		err = db.InsertClient(expectedClient.Name, expectedClient.E4ID, expectedClient.Key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		client, err = db.GetClientByID(expectedClient.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(client, expectedClient) == false {
			t.Errorf("Expected client to be %#v, got %#v", expectedClient, client)
		}
	})

	t.Run("GetClient with unknown id return record not found error", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.GetClientByID([]byte("unknown"))
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

	t.Run("GetTopicKey with unknown topic return record not found error", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.GetTopicKey("unknown")
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("Delete properly delete Client", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		expectedClient := Client{
			ID:   1,
			Name: "someName",
			E4ID: e4.HashIDAlias("someName"),
			Key:  []byte("someKey"),
		}

		err := db.InsertClient(expectedClient.Name, expectedClient.E4ID, expectedClient.Key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		err = db.DeleteClientByID(expectedClient.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		_, err = db.GetClientByID(expectedClient.E4ID)
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("Delete unknown Client return record not found", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		err := db.DeleteClientByID([]byte("unknown"))
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

	t.Run("Delete unknown topicKey returns record not found error", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		err := db.DeleteTopicKey("unknown")
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("CountClients properly count Clients", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		clients := []string{
			"a",
			"b",
			"c",
			"d",
			"e",
		}

		for i, name := range clients {
			c, err := db.CountClients()
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if c != i {
				t.Errorf("Expected count to be %d, got %d", i, c)
			}

			err = db.InsertClient(name, e4.HashIDAlias(name), []byte("key"))
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		}

		for i, name := range clients {
			c, err := db.CountClients()
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if c != len(clients)-i {
				t.Errorf("Expected count to be %d, got %d", len(clients)-i, c)
			}

			err = db.DeleteClientByID(e4.HashIDAlias(name))
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

	t.Run("GetAllClients returns all Clients", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		clients, err := db.GetAllClients()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(clients) != 0 {
			t.Errorf("Expected %d Clients, got %d", 0, len(clients))
		}

		expectedClients := []Client{
			Client{ID: 1, Name: "Client1", E4ID: e4.HashIDAlias("Client1"), Key: []byte("key1")},
			Client{ID: 2, Name: "Client2", E4ID: e4.HashIDAlias("Client2"), Key: []byte("key2")},
			Client{ID: 3, Name: "Client3", E4ID: e4.HashIDAlias("Client3"), Key: []byte("key3")},
		}

		for _, client := range expectedClients {
			err = db.InsertClient(client.Name, client.E4ID, client.Key)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		}

		clients, err = db.GetAllClients()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(clients, expectedClients) == false {
			t.Errorf("Expected clients to be %#v, got %#v", expectedClients, clients)
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
			t.Errorf("Expected clients to be %#v, got %#v", expectedTopics, topics)
		}
	})

	t.Run("LinkClientTopic properly link an Client to a TopicKey", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		client := Client{ID: 1, Name: "i-1", E4ID: e4.HashIDAlias("i-1"), Key: []byte("key")}
		if err := db.InsertClient(client.Name, client.E4ID, client.Key); err != nil {
			t.Fatalf("Failed to insert Client: %v", err)
		}

		topic := TopicKey{ID: 1, Topic: "t-1", Key: []byte("key")}
		if err := db.InsertTopicKey(topic.Topic, topic.Key); err != nil {
			t.Fatalf("Failed to insert TopicKey: %v", err)
		}

		if err := db.LinkClientTopic(client, topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		count, err := db.CountClientsForTopic(topic.Topic)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count to be 1, got %d", count)
		}

		count, err = db.CountTopicsForClientByID(client.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count to be 1, got %d", count)
		}

		topics, err := db.GetTopicsForClientByID(client.E4ID, 0, 10)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(topics, []TopicKey{topic}) == false {
			t.Errorf("Expected topics to be %#v, got %#v", []TopicKey{topic}, topics)
		}

		clients, err := db.GetClientsForTopic(topic.Topic, 0, 10)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(clients, []Client{client}) == false {
			t.Errorf("Expected clients to be %#v, got %#v", []Client{client}, clients)
		}

		if err := db.UnlinkClientTopic(client, topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		topics, err = db.GetTopicsForClientByID(client.E4ID, 0, 10)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(topics) != 0 {
			t.Errorf("Expected no topics, got %#v", topics)
		}

		clients, err = db.GetClientsForTopic(topic.Topic, 0, 10)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(clients) != 0 {
			t.Errorf("Expected no clients, got %#v", clients)
		}

		count, err = db.CountClientsForTopic(topic.Topic)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if count != 0 {
			t.Errorf("Expected count to be 0, got %d", count)
		}

		count, err = db.CountTopicsForClientByID(client.E4ID)
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

		client := Client{Name: "a", E4ID: e4.HashIDAlias("a"), Key: []byte("b")}
		topicKey := TopicKey{Topic: "c", Key: []byte("d")}

		if err := db.LinkClientTopic(client, topicKey); err != ErrClientNoPrimaryKey {
			t.Errorf("Expected error to be %v, got %v", ErrClientNoPrimaryKey, err)
		}

		if err := db.InsertClient(client.Name, client.E4ID, client.Key); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		client.ID = 1

		if err := db.LinkClientTopic(client, topicKey); err != ErrTopicKeyNoPrimaryKey {
			t.Errorf("Expected error to be %v, got %v", ErrTopicKeyNoPrimaryKey, err)
		}
	})

	t.Run("Unlink with unknown records return errors", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		client := Client{Name: "a", E4ID: e4.HashIDAlias("a"), Key: []byte("b")}
		topicKey := TopicKey{Topic: "c", Key: []byte("d")}

		if err := db.UnlinkClientTopic(client, topicKey); err != ErrClientNoPrimaryKey {
			t.Errorf("Expected error to be %v, got %v", ErrClientNoPrimaryKey, err)
		}

		if err := db.InsertClient(client.Name, client.E4ID, client.Key); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		client.ID = 1

		if err := db.UnlinkClientTopic(client, topicKey); err != ErrTopicKeyNoPrimaryKey {
			t.Errorf("Expected error to be %v, got %v", ErrTopicKeyNoPrimaryKey, err)
		}
	})

	t.Run("GetIdsforTopic with unknown topic returns a RecordNotFound error", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.GetClientsForTopic("unknown", 0, 1)
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("GetTopicsForClientByXxx with unknown topic returns a RecordNotFound error", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.GetTopicsForClientByID([]byte("unknown"), 0, 1)
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
		_, err = db.GetTopicsForClientByID([]byte("unknown"), 0, 1)
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("CountClientsForTopic returns a record not found when topic doesn't exists", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.CountClientsForTopic("unknown")
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("CountTopicsForID returns a record not found when topic doesn't exists", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		_, err := db.CountTopicsForClientByID([]byte("unknown"))
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
		}
	})

	t.Run("Migrate on already migrated DB succeeds", func(t *testing.T) {
		db, tearDown := setup(t)
		defer tearDown()

		err := db.Migrate()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
