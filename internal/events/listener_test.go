package events

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
)

func TestListener(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDispatcher := NewMockDispatcher(mockCtrl)

	t.Run("listener register / unregister itself on the dispatcher", func(t *testing.T) {
		mockDispatcher.EXPECT().AddListener(gomock.Any())
		lis := NewListener(mockDispatcher)
		if lis.C() == nil {
			t.Errorf("Expected listener channel to be initialized, got nil")
		}

		if cap(lis.C()) != EventChanBufferSize {
			t.Errorf("Expected listener channel capacity to be %d, got %d", EventChanBufferSize, cap(lis.C()))
		}

		mockDispatcher.EXPECT().RemoveListener(lis)
		err := lis.Close()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Send add event to the listener channel, and discard oldest events when full", func(t *testing.T) {
		EventChanBufferSize = 3 // Smaller channel size for testing

		mockDispatcher.EXPECT().AddListener(gomock.Any())
		lis := NewListener(mockDispatcher)

		if cap(lis.C()) != EventChanBufferSize {
			t.Errorf("Expected listener channel capacity to be %d, got %d", EventChanBufferSize, cap(lis.C()))
		}

		event1 := Event{Type: ClientSubscribed, Source: "client1", Target: "topic1"}
		event2 := Event{Type: ClientSubscribed, Source: "client2", Target: "topic2"}
		event3 := Event{Type: ClientSubscribed, Source: "client3", Target: "topic3"}
		event4 := Event{Type: ClientSubscribed, Source: "client4", Target: "topic4"}

		lis.Send(event1)
		if len(lis.C()) != 1 {
			t.Errorf("Expected listener channel length to be %d, got %d", 1, len(lis.C()))
		}

		select {
		case <-time.After(1 * time.Millisecond):
			t.Errorf("Timeout while waiting for an event")
		case e := <-lis.C():
			if e != event1 {
				t.Errorf("Expected first channel event to be %#v, got %#v", event1, e)
			}
		}

		lis.Send(event1)
		lis.Send(event2)
		lis.Send(event3)
		lis.Send(event4)
		if len(lis.C()) != EventChanBufferSize {
			t.Errorf("Expected listener channel length to be %d, got %d", EventChanBufferSize, len(lis.C()))
		}

		select {
		case <-time.After(1 * time.Millisecond):
			t.Errorf("Timeout while waiting for an event")
		case e := <-lis.C():
			if e != event2 {
				t.Errorf("Expected first channel event to be %#v, got %#v", event2, e)
			}
		}
	})
}
