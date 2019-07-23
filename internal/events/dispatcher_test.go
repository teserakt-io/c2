package events

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/go-kit/kit/log"
)

func TestDispatcher(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	logger := log.NewNopLogger()

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
