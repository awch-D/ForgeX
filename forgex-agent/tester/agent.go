// Package tester implements the Tester Agent.
// It picks up test requests from the event bus, runs builds/tests via MCP tools,
// and reports results back.
package tester

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-mcp/tools"
)

// Agent is the Tester specialist.
type Agent struct {
	bus      *protocol.EventBus
	inbox    <-chan protocol.Message
	registry *tools.Registry
}

// New creates a tester agent.
func New(bus *protocol.EventBus, registry *tools.Registry) *Agent {
	inbox := bus.Subscribe(protocol.RoleTester, 20)
	return &Agent{bus: bus, inbox: inbox, registry: registry}
}

func (a *Agent) Role() protocol.AgentRole { return protocol.RoleTester }

// Run waits for test requests, executes builds/tests, and reports results.
func (a *Agent) Run(ctx context.Context) error {
	for {
		select {
		case msg, ok := <-a.inbox:
			if !ok {
				return nil
			}
			if msg.Type == protocol.MsgTest || msg.Type == protocol.MsgTask {
				a.handleTestRequest(ctx, msg)
			}

		case <-ctx.Done():
			return nil
		}
	}
}

func (a *Agent) handleTestRequest(ctx context.Context, msg protocol.Message) {
	logger.L().Infow("🧪 Tester: received test request", "from", msg.Sender)

	// Step 1: Try to build
	buildResult := a.registry.Execute("run_command", map[string]string{
		"command": "GOWORK=off go build ./... 2>&1",
	})

	buildPassed := buildResult.Success
	var testOutput strings.Builder
	testOutput.WriteString("=== Build ===\n")
	testOutput.WriteString(buildResult.Output)

	// Step 2: Try to run tests (if build passed)
	testsPassed := false
	totalTests := 0
	failedTests := 0

	if buildPassed {
		testResult := a.registry.Execute("run_command", map[string]string{
			"command": "GOWORK=off go test -v ./... 2>&1",
		})
		testOutput.WriteString("\n=== Tests ===\n")
		testOutput.WriteString(testResult.Output)
		testsPassed = testResult.Success

		// Count test lines
		for _, line := range strings.Split(testResult.Output, "\n") {
			if strings.Contains(line, "--- PASS") {
				totalTests++
			} else if strings.Contains(line, "--- FAIL") {
				totalTests++
				failedTests++
			}
		}
	}

	// Step 3: Vet check
	if buildPassed {
		vetResult := a.registry.Execute("run_command", map[string]string{
			"command": "GOWORK=off go vet ./... 2>&1",
		})
		testOutput.WriteString("\n=== Vet ===\n")
		testOutput.WriteString(vetResult.Output)
		if !vetResult.Success {
			testOutput.WriteString("⚠️ go vet found issues\n")
		}
	}

	payload := protocol.TestPayload{
		TaskID:      "test-all",
		Passed:      buildPassed && testsPassed,
		TotalTests:  totalTests,
		FailedTests: failedTests,
		Output:      testOutput.String(),
	}

	payloadJSON, _ := json.Marshal(payload)
	var payloadMap map[string]interface{}
	json.Unmarshal(payloadJSON, &payloadMap)

	a.bus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleTester,
		Receiver: protocol.RoleSupervisor,
		Type:     protocol.MsgTest,
		Payload:  payload,
	})

	status := "PASSED ✅"
	if !payload.Passed {
		status = fmt.Sprintf("FAILED ❌ (%d/%d)", failedTests, totalTests)
	}
	logger.L().Infow("🧪 Tester: report sent", "status", status)
}
