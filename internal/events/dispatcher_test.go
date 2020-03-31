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

package events

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	log "github.com/sirupsen/logrus"
)

func TestDispatcher(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	logger := log.New()
	logger.SetOutput(ioutil.Discard)

	t.Run("Dispatcher register and hold listeners", func(t *testing.T) {
		dispatcher := NewDispatcher(logger)

		lis1 := NewMockListener(mockCtrl)
		lis2 := NewMockListener(mockCtrl)
		lis3 := NewMockListener(mockCtrl)

		dispatcher.AddListener(lis1)
		dispatcher.AddListener(lis2)
		dispatcher.AddListener(lis3)

		expectedListeners := []Listener{lis1, lis2, lis3}

		if reflect.DeepEqual(expectedListeners, dispatcher.Listeners()) == false {
			t.Errorf("Expected listeners to be %#v, got %#v", expectedListeners, dispatcher.Listeners())
		}

		err := dispatcher.RemoveListener(lis2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedListeners = []Listener{lis1, lis3}
		if reflect.DeepEqual(expectedListeners, dispatcher.Listeners()) == false {
			t.Errorf("Expected listeners to be %#v, got %#v", expectedListeners, dispatcher.Listeners())
		}
	})

	t.Run("Removing unknow listener returns an error", func(t *testing.T) {
		dispatcher := NewDispatcher(logger)

		lis := &listener{}
		err := dispatcher.RemoveListener(lis)
		if err != ErrListenerNotFound {
			t.Errorf("Expected err to be %v, got %v", ErrListenerNotFound, err)
		}
	})

	t.Run("Dispatch forward the event to every listeners", func(t *testing.T) {
		dispatcher := NewDispatcher(logger)

		lis1 := NewMockListener(mockCtrl)
		lis2 := NewMockListener(mockCtrl)

		dispatcher.AddListener(lis1)
		dispatcher.AddListener(lis2)

		expectedEvent := Event{
			Type:   ClientSubscribed,
			Source: "client1",
			Target: "topic1",
		}

		lis1.EXPECT().Send(expectedEvent)
		lis2.EXPECT().Send(expectedEvent)

		dispatcher.Dispatch(expectedEvent)
	})
}
