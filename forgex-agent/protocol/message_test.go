package protocol_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
)

func TestEventBus_TargetedDelivery(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()

	coderCh := bus.Subscribe(protocol.RoleCoder, 10)
	testerCh := bus.Subscribe(protocol.RoleTester, 10)

	ctx := context.Background()
	bus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleCoder,
		Type:     protocol.MsgTask,
		Payload:  "coder only",
	})

	// Coder should receive
	select {
	case msg := <-coderCh:
		if msg.Payload != "coder only" {
			t.Errorf("expected 'coder only', got %v", msg.Payload)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("coder did not receive message")
	}

	// Tester should NOT receive
	select {
	case msg := <-testerCh:
		t.Fatalf("tester should not receive targeted coder message, got %v", msg)
	case <-time.After(50 * time.Millisecond):
		// OK
	}
}

func TestEventBus_Broadcast(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()

	coderCh := bus.Subscribe(protocol.RoleCoder, 10)
	testerCh := bus.Subscribe(protocol.RoleTester, 10)

	ctx := context.Background()
	bus.Publish(ctx, protocol.Message{
		Sender:  protocol.RoleSupervisor,
		Type:    protocol.MsgStatus,
		Payload: "broadcast",
	})

	// Both should receive
	for name, ch := range map[string]<-chan protocol.Message{"coder": coderCh, "tester": testerCh} {
		select {
		case msg := <-ch:
			if msg.Payload != "broadcast" {
				t.Errorf("%s: expected 'broadcast', got %v", name, msg.Payload)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("%s did not receive broadcast", name)
		}
	}
}

func TestEventBus_SubscribeAll(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()

	allCh := bus.SubscribeAll(10)
	_ = bus.Subscribe(protocol.RoleCoder, 10)

	ctx := context.Background()
	bus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleCoder,
		Type:     protocol.MsgTask,
		Payload:  "targeted msg",
	})

	select {
	case msg := <-allCh:
		if msg.Payload != "targeted msg" {
			t.Errorf("expected 'targeted msg', got %v", msg.Payload)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("SubscribeAll did not receive message")
	}
}

func TestEventBus_Timestamp(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()

	ch := bus.Subscribe(protocol.RoleCoder, 10)
	before := time.Now()

	ctx := context.Background()
	bus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleCoder,
		Type:     protocol.MsgTask,
	})

	msg := <-ch
	if msg.Timestamp.Before(before) {
		t.Errorf("timestamp %v should be after %v", msg.Timestamp, before)
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()

	ch := bus.Subscribe(protocol.RoleCoder, 100)
	ctx := context.Background()

	var wg sync.WaitGroup
	n := 50
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(id int) {
			defer wg.Done()
			bus.Publish(ctx, protocol.Message{
				Sender:   protocol.RoleSupervisor,
				Receiver: protocol.RoleCoder,
				Type:     protocol.MsgTask,
				Payload:  id,
			})
		}(i)
	}
	wg.Wait()

	received := 0
	for {
		select {
		case <-ch:
			received++
		case <-time.After(100 * time.Millisecond):
			goto done
		}
	}
done:
	if received != n {
		t.Errorf("expected %d messages, got %d", n, received)
	}
}

func TestEventBus_CloseIdempotent(t *testing.T) {
	bus := protocol.NewEventBus()
	bus.Close()
	bus.Close() // Should not panic
}

func TestEventBus_PublishAfterClose(t *testing.T) {
	bus := protocol.NewEventBus()
	_ = bus.Subscribe(protocol.RoleCoder, 10)
	bus.Close()

	// Should not panic
	bus.Publish(context.Background(), protocol.Message{
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleCoder,
		Type:     protocol.MsgTask,
	})
}
