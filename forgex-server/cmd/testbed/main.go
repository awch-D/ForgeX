package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
	"github.com/awch-D/ForgeX/forgex-server/bus"
)

func main() {
	eventBus := protocol.NewEventBus()
	defer eventBus.Close()

	hub := bus.NewHub(eventBus)
	go hub.Run()

	http.HandleFunc("/ws", hub.HandleWebSocket)
	fmt.Println("Server starts at :8080")
	go http.ListenAndServe(":8080", nil)

	time.Sleep(2 * time.Second) // wait for react to align

	ctx := context.Background()

	// 1. Send task
	eventBus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleCoder,
		Type:     protocol.MsgTask,
		Payload: protocol.TaskPayload{
			TaskID:      "test-1",
			Description: "Make a nice glowing button for the dashboard",
		},
	})

	time.Sleep(1 * time.Second)

	// 2. Send status from developer
	eventBus.Publish(ctx, protocol.Message{
		Sender:  protocol.RoleCoder,
		Type:    protocol.MsgStatus,
		Payload: "I'm looking at lucide-react right now and setting up Tailwind",
	})

	time.Sleep(1 * time.Second)

	// 3. Send test execution
	eventBus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleTester,
		Receiver: protocol.RoleSupervisor,
		Type:     protocol.MsgTest,
		Payload: protocol.TestPayload{
			TaskID:      "test-1",
			Passed:      true,
			TotalTests:  5,
			FailedTests: 0,
		},
	})

	select {}
}
