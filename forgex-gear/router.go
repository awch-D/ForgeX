// Package gear provides the intelligent task routing engine.
// It evaluates task complexity and decides between single-agent (L1/L2)
// and multi-agent (L3/L4) execution strategies.
package gear

import (
	"strings"

	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-intent/parser"
)

// Level represents the task complexity level.
type Level int

const (
	L1Simple   Level = 1 // Single file, straightforward
	L2Medium   Level = 2 // Multiple files, clear structure
	L3Complex  Level = 3 // Multiple modules, needs decomposition
	L4Advanced Level = 4 // System-level, needs architecture design
)

func (l Level) String() string {
	switch l {
	case L1Simple:
		return "L1-Simple"
	case L2Medium:
		return "L2-Medium"
	case L3Complex:
		return "L3-Complex"
	case L4Advanced:
		return "L4-Advanced"
	default:
		return "Unknown"
	}
}

// NeedsMultiAgent returns true if the level requires multi-agent coordination.
func (l Level) NeedsMultiAgent() bool {
	return l >= L3Complex
}

// Evaluate analyzes a task and returns its complexity level.
func Evaluate(analysis *parser.TaskAnalysis) Level {
	score := 0

	// Check number of planned files
	fileCount := len(analysis.ExecutionPlan)
	if fileCount <= 2 {
		score += 1
	} else if fileCount <= 5 {
		score += 2
	} else {
		score += 3
	}

	// Check tech stack complexity
	if len(analysis.TechStack) > 3 {
		score += 2
	} else if len(analysis.TechStack) > 1 {
		score += 1
	}

	// Keyword complexity indicators
	desc := strings.ToLower(analysis.CoreIntent)
	complexityKeywords := []string{
		"微服务", "microservice", "分布式", "distributed",
		"认证", "鉴权", "auth", "jwt", "oauth",
		"数据库", "database", "缓存", "cache",
		"并发", "concurrent", "异步", "async",
		"api", "rest", "grpc", "websocket",
	}

	keywordHits := 0
	for _, kw := range complexityKeywords {
		if strings.Contains(desc, kw) {
			keywordHits++
		}
	}
	if keywordHits >= 4 {
		score += 3
	} else if keywordHits >= 2 {
		score += 2
	} else if keywordHits >= 1 {
		score += 1
	}

	var level Level
	switch {
	case score <= 2:
		level = L1Simple
	case score <= 4:
		level = L2Medium
	case score <= 6:
		level = L3Complex
	default:
		level = L4Advanced
	}

	logger.L().Infow("⚙️ Gear evaluation",
		"score", score,
		"level", level.String(),
		"multi_agent", level.NeedsMultiAgent(),
		"file_count", fileCount,
		"keyword_hits", keywordHits,
	)

	return level
}
