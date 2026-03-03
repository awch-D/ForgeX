// Package evolution provides an automated code quality evaluation loop.
// After code generation, it runs compilation and tests to score the output,
// and decides whether the agent should retry with error context.
package evolution

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/awch-D/ForgeX/forgex-core/logger"
)

// Score holds the evaluation results for generated code.
type Score struct {
	CompilePass bool    `json:"compile_pass"`
	TestPass    bool    `json:"test_pass"`
	TestTotal   int     `json:"test_total"`
	TestFailed  int     `json:"test_failed"`
	Total       float64 `json:"total"` // 0.0 ~ 1.0
	Errors      string  `json:"errors,omitempty"`
}

// Evolver evaluates generated code quality and decides on retries.
type Evolver struct {
	MaxRetries int
	Threshold  float64 // minimum acceptable total score (0~1)
}

// NewEvolver creates an evolver with sensible defaults.
func NewEvolver() *Evolver {
	return &Evolver{
		MaxRetries: 2,
		Threshold:  0.6,
	}
}

// Evaluate runs build and test commands in the given directory and returns a score.
func (e *Evolver) Evaluate(dir string) Score {
	score := Score{}

	// Step 1: Try to compile
	compileErr := e.tryCompile(dir)
	if compileErr == "" {
		score.CompilePass = true
	} else {
		score.Errors = compileErr
		score.Total = 0.0
		return score
	}

	// Step 2: Try to run tests
	testOut, testErr := e.tryTest(dir)
	if testErr == "" {
		score.TestPass = true
	} else {
		score.Errors = testErr
	}

	// Parse test results
	score.TestTotal, score.TestFailed = parseTestOutput(testOut)

	// Calculate total score
	compileScore := 0.0
	if score.CompilePass {
		compileScore = 1.0
	}

	testScore := 0.0
	if score.TestTotal > 0 {
		testScore = float64(score.TestTotal-score.TestFailed) / float64(score.TestTotal)
	} else if score.CompilePass {
		testScore = 0.5 // compiled but no tests found
	}

	// Weighted: compile 40%, tests 60%
	score.Total = compileScore*0.4 + testScore*0.6

	logger.L().Infow("📊 Evolution score",
		"compile", score.CompilePass,
		"test_pass", score.TestPass,
		"test_total", score.TestTotal,
		"test_failed", score.TestFailed,
		"total", fmt.Sprintf("%.2f", score.Total),
	)

	return score
}

// ShouldRetry returns true if the score is below the threshold.
func (e *Evolver) ShouldRetry(s Score) bool {
	return s.Total < e.Threshold
}

// BuildRetryPrompt constructs a prompt that tells the agent what went wrong.
func (e *Evolver) BuildRetryPrompt(s Score) string {
	var sb strings.Builder
	sb.WriteString("The previously generated code has quality issues. Please fix the following problems:\n\n")

	if !s.CompilePass {
		sb.WriteString("❌ COMPILATION FAILED:\n")
		sb.WriteString(s.Errors)
		sb.WriteString("\n\n")
	}

	if !s.TestPass && s.CompilePass {
		sb.WriteString(fmt.Sprintf("❌ TEST FAILURES: %d/%d tests failed\n", s.TestFailed, s.TestTotal))
		sb.WriteString(s.Errors)
		sb.WriteString("\n\n")
	}

	sb.WriteString("Please analyze the errors above and fix the code. Output the corrected version.")
	return sb.String()
}

func (e *Evolver) tryCompile(dir string) string {
	// Detect language by checking for common build files
	if hasFile(dir, "go.mod") {
		return runCmd(dir, "go", "build", "./...")
	}
	if hasFile(dir, "package.json") {
		return runCmd(dir, "npm", "run", "build")
	}
	if hasFile(dir, "Cargo.toml") {
		return runCmd(dir, "cargo", "build")
	}
	// For scripts (Python etc.), just check syntax
	if hasFile(dir, "*.py") {
		return runCmd(dir, "python3", "-m", "py_compile", findFirst(dir, "*.py"))
	}
	return "" // No build system detected, assume OK
}

func (e *Evolver) tryTest(dir string) (output string, errMsg string) {
	if hasFile(dir, "go.mod") {
		out, err := runCmdWithOutput(dir, "go", "test", "-v", "./...")
		if err != "" {
			return out, err
		}
		return out, ""
	}
	if hasFile(dir, "package.json") {
		out, err := runCmdWithOutput(dir, "npm", "test")
		if err != "" {
			return out, err
		}
		return out, ""
	}
	return "", "" // No test system detected
}

func runCmd(dir string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output)
	}
	return ""
}

func runCmdWithOutput(dir string, name string, args ...string) (string, string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Sprintf("%s\n%s", string(output), err.Error())
	}
	return string(output), ""
}

func hasFile(dir, pattern string) bool {
	matches, _ := filepath.Glob(filepath.Join(dir, pattern))
	return len(matches) > 0
}

func findFirst(dir, pattern string) string {
	matches, _ := filepath.Glob(filepath.Join(dir, pattern))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func parseTestOutput(output string) (total, failed int) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "---") {
			if strings.Contains(line, "PASS") {
				total++
			} else if strings.Contains(line, "FAIL") {
				total++
				failed++
			}
		}
	}
	return total, failed
}
