package models

// import (
// 	"bytes"
// 	"crypto/rand"
// 	"fmt"
// 	"os"
// 	"testing"

// 	"github.com/go-kit/kit/log"
// 	"github.com/jinzhu/gorm"
// 	_ "github.com/jinzhu/gorm/dialects/postgres"
// 	_ "github.com/jinzhu/gorm/dialects/sqlite"

// 	e4 "gitlab.com/teserakt/e4common"
// 	e4test "gitlab.com/teserakt/test-common"
// )

// func testInitializeFakeC2(db *gorm.DB) C2 {
// 	var c2 C2
// 	keyenckey := e4.HashPwd("unittests")
// 	copy(c2.keyenckey[:], keyenckey[:])
// 	c2.logger = log.NewJSONLogger(os.Stderr)
// 	c2.db = db

// 	return c2
// }

// // TestPlaceHolder No unit tests yet, but we should have them.
// func TestM2MSQLite(t *testing.T) {
// 	dbPath := e4test.GetRandomDBName()

// 	fmt.Fprintf(os.Stderr, "Database Path: %s\n", dbPath)

// 	db, err := gorm.Open("sqlite3", dbPath)
// 	if err != nil {
// 		t.Errorf("Error: %s", err)
// 	}

// 	c2 := testInitializeFakeC2(db)

// 	// close and delete the database on exit.
// 	defer func() {
// 		c2.db.Close()
// 		os.Remove(dbPath)
// 	}()

// 	// Initialize DB
// 	if err := c2.dbInitialize(); err != nil {
// 		t.Errorf("Error: %s", err)
// 	}

// 	const IDS int = 50
// 	const TOPICS int = 10
// 	const INSERTLATERIDS int = 7
// 	const INSERTLATERTOPICS int = 5

// 	var ids [IDS][]byte
// 	var idkeys [IDS][]byte

// 	for i := 0; i < IDS; i++ {
// 		ids[i], err = e4test.GenerateID()
// 		if err != nil {
// 			t.Errorf("Error: %s", err)
// 		}
// 		idkeys[i], err = e4test.GenerateKey()
// 		if err != nil {
// 			t.Errorf("Error: %s", err)
// 		}
// 	}

// 	var topics [TOPICS]string
// 	var topickeys [TOPICS][]byte
// 	for i := 0; i < TOPICS; i++ {
// 		topics[i] = fmt.Sprintf("testtopic%d", i)
// 		topickeys[i], err = e4test.GenerateKey()
// 		if err != nil {
// 			t.Errorf("Error: %s", err)
// 		}
// 	}

// 	// insert all but INSERTLATERIDS of the id keys
// 	for i := 0; i < IDS-INSERTLATERIDS; i++ {
// 		c2.dbInsertIDKey(ids[i], idkeys[i])
// 	}

// 	// insert all but 1 of the topics:
// 	for i := 0; i < TOPICS-INSERTLATERTOPICS; i++ {
// 		c2.dbInsertTopicKey(topics[i], topickeys[i])
// 	}

// 	// check the database fetching APIs correctly return valid
// 	// values
// 	for i := 0; i < IDS-INSERTLATERIDS; i++ {
// 		idkey, err := c2.dbGetIDKey(ids[i])
// 		if err != nil {
// 			t.Errorf("Error: %s", err)
// 		}
// 		if !bytes.Equal(idkey, idkeys[i]) {
// 			t.Error("Topic key sorted doesn't match inserted value.")
// 		}

// 	}
// 	for i := 0; i < TOPICS-INSERTLATERTOPICS; i++ {
// 		topickey, err := c2.dbGetTopicKey(topics[i])
// 		if err != nil {
// 			t.Errorf("Error: %s", err)
// 		}
// 		if !bytes.Equal(topickey, topickeys[i]) {
// 			t.Error("Topic key sorted doesn't match inserted value.")
// 		}

// 	}

// 	// check we have valid-looking data.
// 	rows, err := c2.db.Raw("SELECT e4_id, key from id_keys").Rows()
// 	if err != nil {
// 		t.Errorf("Error: %s", err)
// 	}

// 	for rows.Next() {
// 		var E4ID []byte
// 		var KeyEncrypted []byte
// 		var KeyDecrypted []byte
// 		found := false

// 		rows.Scan(&E4ID, &KeyEncrypted)

// 		KeyDecrypted, err := e4.Decrypt(c2.keyenckey[:], nil, KeyEncrypted)
// 		if err != nil {
// 			t.Errorf("Error: %s", err)
// 		}

// 		// TODO: antony needs to learn better golang
// 		for i := 0; i < IDS-INSERTLATERIDS; i++ {
// 			if bytes.Equal(idkeys[i], KeyDecrypted) && bytes.Equal(ids[i], E4ID) {
// 				found = true
// 				break
// 			}
// 		}

// 		if found == false {
// 			t.Errorf("Cannot find inserted data from source data")
// 		}

// 	}
// 	rows.Close()

// 	var randombyte [1]byte
// 	if _, err = rand.Reader.Read(randombyte[:]); err != nil {
// 		t.Errorf("Error: %s", err)
// 	}
// 	rtIdx := int(randombyte[0]) % (TOPICS - INSERTLATERTOPICS)
// 	randomtopic := topics[rtIdx]

// 	linkedCount := 0

// 	// link the even ids to this topic.
// 	for i := 0; i < IDS-INSERTLATERIDS; i++ {
// 		if i%2 == 0 {
// 			c2.dbLinkIDTopic(ids[i], randomtopic)
// 			linkedCount++
// 		}
// 	}
// 	// look across the m2m relation:
// 	// select e4_id, key FROM id_keys INNER JOIN idkeys_topickeys on id_keys.id=idkeys_topickeys.id_key_id where idkeys_topickeys.topic_key_id=1;
// 	m2msql := fmt.Sprintf("select id, e4_id, key FROM id_keys INNER JOIN idkeys_topickeys on id_keys.id=idkeys_topickeys.id_key_id where idkeys_topickeys.topic_key_id=%d", rtIdx+1)
// 	rows, err = c2.db.Raw(m2msql).Rows()
// 	if err != nil {
// 		t.Errorf("Error: %s", err)
// 	}
// 	linkedCountCheck := 0

// 	for rows.Next() {
// 		var id int
// 		var E4ID []byte
// 		var KeyEncrypted []byte
// 		var KeyDecrypted []byte

// 		rows.Scan(&id, &E4ID, &KeyEncrypted)

// 		KeyDecrypted, err = e4.Decrypt(c2.keyenckey[:], nil, KeyEncrypted)
// 		if err != nil {
// 			t.Errorf("Error: %s", err)
// 		}

// 		if !bytes.Equal(idkeys[id-1], KeyDecrypted) || !bytes.Equal(ids[id-1], E4ID) {
// 			t.Errorf("What should be inserted across this relation hasn't worked.")
// 		}
// 		linkedCountCheck++
// 	}
// 	rows.Close()

// 	if linkedCountCheck != linkedCount {
// 		t.Error("Didn't find the same number of links as we inserted.")
// 	}

// 	// also check via link apis
// 	linkedCountAPICheck, err := c2.dbCountIDsForTopic(randomtopic)
// 	if err != nil {
// 		t.Errorf("Failure calling countIDsForTopic: %s", err)
// 	}
// 	if linkedCountAPICheck != linkedCount {
// 		t.Error("Didn't find the same number of links as we inserted.")
// 	}

// 	if _, err = rand.Reader.Read(randombyte[:]); err != nil {
// 		t.Errorf("Error: %s", err)
// 	}
// 	riIdx := int(randombyte[0]) % TOPICS
// 	randomid := ids[riIdx]

// 	linkedCount = 0
// 	linkedCountCheck = 0

// 	// link the odd topics to this ID:
// 	for i := 0; i < TOPICS-INSERTLATERTOPICS; i++ {
// 		if i%2 == 1 && rtIdx != i {
// 			c2.dbLinkIDTopic(randomid, topics[i])
// 			linkedCount++
// 		}
// 	}

// 	// break the topic-id-links we previously added
// 	// but do it by removing the random topic (check the m2m is cleared up)
// 	c2.dbDeleteTopicKey(randomtopic)

// 	// insert the non-inserted topics and ids.
// 	for i := IDS - INSERTLATERIDS; i < IDS; i++ {
// 		c2.dbInsertIDKey(ids[i], idkeys[i])
// 	}

// 	// insert all but 1 of the topics:
// 	for i := TOPICS - INSERTLATERTOPICS; i < TOPICS; i++ {
// 		c2.dbInsertTopicKey(topics[i], topickeys[i])
// 	}

// 	// check we find nothing across the m2m with our random topic:
// 	m2msql = fmt.Sprintf("select id, e4_id, key FROM id_keys INNER JOIN idkeys_topickeys on id_keys.id=idkeys_topickeys.id_key_id where idkeys_topickeys.topic_key_id=%d", rtIdx+1)
// 	resultrows := 0
// 	c2.db.Raw(m2msql).Count(&resultrows)
// 	if resultrows != 0 {
// 		t.Errorf("Rows returned!")
// 	}

// 	// now check our ID->topics insert survived those manipulations
// 	m2msql = fmt.Sprintf("select id, topic, key FROM topic_keys INNER JOIN idkeys_topickeys on topic_keys.id=idkeys_topickeys.topic_key_id where idkeys_topickeys.id_key_id=%d", riIdx+1)
// 	m2mresult := c2.db.Raw(m2msql)
// 	rows, err = m2mresult.Rows()
// 	if err != nil {
// 		t.Errorf("Error: %s", err)
// 	}
// 	for rows.Next() {
// 		var id int
// 		var Topic string
// 		var KeyEncrypted []byte
// 		var KeyDecrypted []byte

// 		rows.Scan(&id, &Topic, &KeyEncrypted)

// 		KeyDecrypted, err = e4.Decrypt(c2.keyenckey[:], nil, KeyEncrypted)
// 		if err != nil {
// 			t.Errorf("Error: %s", err)
// 		}

// 		if !bytes.Equal(topickeys[id-1], KeyDecrypted) || topics[id-1] != Topic {
// 			t.Errorf("What should be inserted across this relation hasn't worked.")
// 		}
// 		linkedCountCheck++
// 	}
// 	rows.Close()

// 	if linkedCountCheck != linkedCount {
// 		t.Error("Didn't find the same number of links as we inserted.")
// 	}

// 	// also check via link apis
// 	linkedCountAPICheck, err = c2.dbCountTopicsForID(randomid)
// 	if err != nil {
// 		t.Errorf("Failure calling countIDsForTopic: %s", err)
// 	}
// 	if linkedCountAPICheck != linkedCount {
// 		t.Error("Didn't find the same number of links as we inserted.")
// 	}

// 	// remove everything
// 	for i := 0; i < TOPICS; i++ {
// 		c2.dbDeleteTopicKey(topics[i])
// 	}
// 	for i := 0; i < IDS; i++ {
// 		c2.dbDeleteIDKey(ids[i])
// 	}
// }
