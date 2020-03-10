package commands

import (
	reflect "reflect"
	"testing"
)

func TestE4Command(t *testing.T) {
	t.Run("Type returns the expected e4.Command", func(t *testing.T) {
		cmdID := byte(0x01)
		cmd := e4Command([]byte{cmdID})
		cmdType, err := cmd.Type()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if g, w := cmdType, cmdID; g != w {
			t.Errorf("Invalid type, got %v, want %v", g, w)
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
		cmdID := byte(0x01)
		cmd := e4Command([]byte{cmdID})

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
		cmdID := byte(0x01)
		expectedContent := []byte{0x02, 0x03}
		cmd := e4Command(append([]byte{cmdID}, expectedContent...))

		content, err := cmd.Content()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

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
}
