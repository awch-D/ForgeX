// Package display provides a terminal-native event renderer for ForgeX.
// It subscribes to the EventBus and prints agent events with color-coded
// role labels, replacing the need for a separate Web Dashboard.
package display

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pterm/pterm"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
)

// rolePrinter maps agent roles to colored pterm prefixes.
var rolePrinter = map[protocol.AgentRole]*pterm.PrefixPrinter{
	protocol.RoleSupervisor: {
		MessageStyle: pterm.NewStyle(pterm.FgWhite),
		Prefix:       pterm.Prefix{Text: " SUP ", Style: pterm.NewStyle(pterm.BgMagenta, pterm.FgWhite, pterm.Bold)},
	},
	protocol.RoleCoder: {
		MessageStyle: pterm.NewStyle(pterm.FgWhite),
		Prefix:       pterm.Prefix{Text: " DEV ", Style: pterm.NewStyle(pterm.BgCyan, pterm.FgWhite, pterm.Bold)},
	},
	protocol.RoleTester: {
		MessageStyle: pterm.NewStyle(pterm.FgWhite),
		Prefix:       pterm.Prefix{Text: " TST ", Style: pterm.NewStyle(pterm.BgRed, pterm.FgWhite, pterm.Bold)},
	},
	protocol.RoleReviewer: {
		MessageStyle: pterm.NewStyle(pterm.FgWhite),
		Prefix:       pterm.Prefix{Text: " REV ", Style: pterm.NewStyle(pterm.BgYellow, pterm.FgBlack, pterm.Bold)},
	},
}

// StartLiveMonitor subscribes to the EventBus and renders events in the terminal.
// It runs in a goroutine and returns immediately.
func StartLiveMonitor(eventBus *protocol.EventBus) {
	inbox := eventBus.SubscribeAll(100)

	go func() {
		for msg := range inbox {
			renderEvent(msg)
		}
	}()
}

func renderEvent(msg protocol.Message) {
	timestamp := pterm.FgGray.Sprintf("[%s]", time.Now().Format("15:04:05"))

	printer, ok := rolePrinter[msg.Sender]
	if !ok {
		printer = &pterm.PrefixPrinter{
			MessageStyle: pterm.NewStyle(pterm.FgWhite),
			Prefix:       pterm.Prefix{Text: " SYS ", Style: pterm.NewStyle(pterm.BgDarkGray, pterm.FgWhite)},
		}
	}

	target := pterm.FgDarkGray.Sprint("→ broadcast")
	if msg.Receiver != "" {
		target = pterm.FgDarkGray.Sprintf("→ %s", msg.Receiver)
	}

	body := formatPayload(msg)

	// Print: [15:04:05]  SUP  → coder  task  Assigned: xxx
	msgType := pterm.FgDarkGray.Sprintf("[%s]", msg.Type)
	printer.Printfln("%s %s %s %s", timestamp, target, msgType, body)
}

func formatPayload(msg protocol.Message) string {
	switch msg.Type {
	case protocol.MsgTask:
		payloadJSON, _ := json.Marshal(msg.Payload)
		var tp protocol.TaskPayload
		if json.Unmarshal(payloadJSON, &tp) == nil {
			return pterm.FgLightCyan.Sprintf("📋 %s", tp.Description)
		}

	case protocol.MsgResult:
		payloadJSON, _ := json.Marshal(msg.Payload)
		var rp protocol.ResultPayload
		if json.Unmarshal(payloadJSON, &rp) == nil {
			if rp.Success {
				return pterm.FgLightGreen.Sprintf("✅ %s", rp.Summary)
			}
			return pterm.FgLightRed.Sprintf("❌ %s", rp.Error)
		}

	case protocol.MsgTest:
		payloadJSON, _ := json.Marshal(msg.Payload)
		var tp protocol.TestPayload
		if json.Unmarshal(payloadJSON, &tp) == nil {
			if tp.Passed {
				return pterm.FgLightGreen.Sprintf("✅ 测试通过 (%d/%d)", tp.TotalTests-tp.FailedTests, tp.TotalTests)
			}
			return pterm.FgLightRed.Sprintf("❌ 测试失败 (%d/%d failed)", tp.FailedTests, tp.TotalTests)
		}

	case protocol.MsgStatus:
		if s, ok := msg.Payload.(string); ok {
			return pterm.FgLightYellow.Sprint(s)
		}

	case protocol.MsgReview:
		payloadJSON, _ := json.Marshal(msg.Payload)
		var rp protocol.ReviewPayload
		if json.Unmarshal(payloadJSON, &rp) == nil {
			if rp.Passed {
				return pterm.FgLightGreen.Sprintf("✅ 审查通过 (%d/100): %s", rp.Score, truncate(rp.Feedback, 80))
			}
			return pterm.FgLightYellow.Sprintf("⚠️ 审查建议 (%d/100): %s", rp.Score, truncate(rp.Feedback, 80))
		}

	case protocol.MsgComplete:
		return pterm.FgLightGreen.Sprint("🏁 任务完成")
	}

	// Fallback: show raw JSON
	raw, _ := json.Marshal(msg.Payload)
	return string(raw)
}

// truncate shortens s to max runes, appending "…" if cut.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// PrintCostSummary prints a styled cost summary bar.
func PrintCostSummary() {
	tokens, usd := cost.Global().Summary()
	fmt.Println()
	pterm.DefaultBox.
		WithTitle(pterm.LightCyan("💰 成本汇总")).
		WithTitleTopCenter().
		WithBoxStyle(pterm.NewStyle(pterm.FgDarkGray)).
		Println(fmt.Sprintf(
			"%s %s    %s %s",
			pterm.FgYellow.Sprint("Token:"),
			pterm.Bold.Sprintf("%d", tokens),
			pterm.FgGreen.Sprint("花费:"),
			pterm.Bold.Sprintf("$%.4f", usd),
		))
}
