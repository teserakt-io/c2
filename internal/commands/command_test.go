package commands

import (
	reflect "reflect"
	"testing"

	e4 "github.com/teserakt-io/e4go"
	e4crypto "github.com/teserakt-io/e4go/crypto"
)

func TestE4Command(t *testing.T) {
	t.Run("Type returns the expected e4.Command", func(t *testing.T) {
		cmd := e4Command([]byte{0x01})
		cmdType, err := cmd.Type()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if cmdType != e4.Command(0x01) {
			t.Errorf("Expected type to be %v, got %v", e4.Command(0x01), cmdType)
		}
	})

	t.Run("Type returns an error when command is empty", func(t *testing.T) {
		cmd := e4Command{}
		_, err := cmd.Type()
		if err != ErrEmptyCommand {
			t.Errorf("Expected error to be %v, got %v", ErrEmptyCommand, err)
		}
	})

	t.Run("Content with no content check", func(t *testing.T) {
		cmd := e4Command([]byte{0x01})

		content, err := cmd.Content()
		if err != nil {
			t.Errorf("Expected some output, got %v", err)
		}

		expectedContent := []byte{}
		if reflect.DeepEqual(content, expectedContent) == false {
			t.Errorf("Expected content to be %v, got %v", expectedContent, content)
		}
	})

	t.Run("Content returns the expected content", func(t *testing.T) {
		cmd := e4Command([]byte{0x01, 0x02, 0x03})

		content, err := cmd.Content()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedContent := []byte{0x02, 0x03}
		if reflect.DeepEqual(content, expectedContent) == false {
			t.Errorf("Expected content to be %v, got %v", expectedContent, content)
		}
	})

	t.Run("Content returns an error when command is empty", func(t *testing.T) {
		cmd := e4Command{}
		_, err := cmd.Content()
		if err != ErrEmptyCommand {
			t.Errorf("Expected error to be %v, got %v", ErrEmptyCommand, err)
		}
	})

	t.Run("Protect properly encrypt the command", func(t *testing.T) {
		cmd := e4Command([]byte{0x01, 0x02, 0x03})

		key := newKey(t)

		payload, err := cmd.Protect(key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(payload, cmd) == true {
			t.Errorf("Expected payload to be different from original command")
		}

		unprotected, err := e4crypto.UnprotectSymKey(payload, key)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(unprotected, []byte(cmd)) == false {
			t.Errorf("Expected to retrieve original command %v, got %v", cmd, unprotected)
		}
	})

	t.Run("Protect with invalid key returns error", func(t *testing.T) {
		cmd := e4Command([]byte{0x01, 0x02, 0x03})

		_, err := cmd.Protect([]byte("invalid"))
		if err == nil {
			t.Errorf("Expected an error, got nil")
		}
	})
}
