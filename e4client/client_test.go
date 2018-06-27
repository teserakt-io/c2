package e4client

import (
	"testing"
)

func TestWriteRead(t *testing.T) {
	filePath := "./e4clienttest"

	c := NewClient(nil, nil, filePath)
	err := writeGob(filePath, c)
	if err != nil {
		t.Fatalf("save failed: %s", err)
	}

	cc, err := LoadClient(filePath)
	if err != nil {
		t.Fatalf("client loading failed: %s", err)
	}

	if string(cc.Id) != string(c.Id) {
		t.Fatal("id doesnt match")
	}
	if string(cc.Key) != string(c.Key) {
		t.Fatal("key doesnt match")
	}
	if cc.FilePath != c.FilePath {
		t.Fatal("filepath doesnt match")
	}
}
