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
// Scoring dimensions:
//  1. Execution plan size (number of steps/files to create)
//  2. Tech stack breadth
//  3. Keyword complexity indicators
//  4. Files to modify count (existing codebase impact)
//  5. LLM's own estimation (as reference signal)
func Evaluate(analysis *parser.TaskAnalysis) Level {
	score := 0

	// Dimension 1: Execution plan size
	fileCount := len(analysis.ExecutionPlan)
	switch {
	case fileCount <= 2:
		score += 1
	case fileCount <= 5:
		score += 2
	case fileCount <= 10:
		score += 3
	default:
		score += 4
	}

	// Dimension 2: Tech stack breadth
	switch {
	case len(analysis.TechStack) > 4:
		score += 3
	case len(analysis.TechStack) > 2:
		score += 2
	case len(analysis.TechStack) > 0:
		score += 1
	}

	// Dimension 3: Keyword complexity indicators
	desc := strings.ToLower(analysis.CoreIntent)
	complexityKeywords := map[string]int{
		// High complexity (weight 2)
		"微服务": 2, "microservice": 2, "分布式": 2, "distributed": 2,
		"并发": 2, "concurrent": 2, "实时": 2, "realtime": 2,
		"grpc": 2, "websocket": 2,
		// Medium complexity (weight 1)
		"认证": 1, "鉴权": 1, "auth": 1, "jwt": 1, "oauth": 1,
		"数据库": 1, "database": 1, "缓存": 1, "cache": 1,
		"异步": 1, "async": 1,
		"api": 1, "rest": 1, "crud": 1,
	}

	keywordScore := 0
	keywordHits := 0
	for kw, weight := range complexityKeywords {
		if strings.Contains(desc, kw) {
			keywordScore += weight
			keywordHits++
		}
	}
	switch {
	case keywordScore >= 6:
		score += 4
	case keywordScore >= 4:
		score += 3
	case keywordScore >= 2:
		score += 2
	case keywordScore >= 1:
		score += 1
	}

	// Dimension 4: Existing codebase impact (files to modify)
	modifyCount := len(analysis.FilesToModify)
	switch {
	case modifyCount > 10:
		score += 3
	case modifyCount > 5:
		score += 2
	case modifyCount > 0:
		score += 1
	}

	// Dimension 5: LLM's own estimation (as calibration signal)
	if analysis.EstimatedLevel >= 3 {
		score += 2
	} else if analysis.EstimatedLevel >= 2 {
		score += 1
	}

	// Map score to level
	var level Level
	switch {
	case score <= 3:
		level = L1Simple
	case score <= 6:
		level = L2Medium
	case score <= 10:
		level = L3Complex
	default:
		level = L4Advanced
	}

	logger.L().Infow("⚙️ Gear evaluation",
		"score", score,
		"level", level.String(),
		"multi_agent", level.NeedsMultiAgent(),
		"file_count", fileCount,
		"modify_count", modifyCount,
		"keyword_hits", keywordHits,
		"keyword_score", keywordScore,
		"llm_estimated", analysis.EstimatedLevel,
	)

	return level
}
