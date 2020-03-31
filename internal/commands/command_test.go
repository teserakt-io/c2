// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
