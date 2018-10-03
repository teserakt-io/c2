package e4client

import (
	"testing"

	e4 "teserakt/e4go/pkg/e4common"
)

func TestWriteRead(t *testing.T) {
	filePath := "./e4clienttest"

	c := NewClientPretty("someid", "somepwd", filePath)

	err := c.SetTopicKey(e4.RandomKey(), e4.HashTopic("meh"))
	if err != nil {
		t.Fatalf("SetTopicKey failed: %s", err)
	}

	if len(c.Topickeys) != 1 {
		t.Fatalf("invalid number of topic keys: %d vs 1 expected", len(c.Topickeys))
	}

	// state should be saved here
	err = c.ResetTopics()

	if err != nil {
		t.Fatalf("save failed: %s", err)
	}

	cc, err := LoadClient(filePath)
	if err != nil {
		t.Fatalf("client loading failed: %s", err)
	}

	if string(cc.ID) != string(c.ID) {
		t.Fatal("id doesnt match")
	}
	if string(cc.Key) != string(c.Key) {
		t.Fatal("key doesnt match")
	}
	if cc.FilePath != c.FilePath {
		t.Fatal("filepath doesnt match")
	}
	// check that topickeys on disk was changed after ResetTopics
	if len(cc.Topickeys) != 0 {
		t.Fatalf("invalid number of topic keys: %d vs 0 expected", len(cc.Topickeys))
	}
}
