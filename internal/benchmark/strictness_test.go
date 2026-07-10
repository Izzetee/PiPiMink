package benchmark

import (
	"strings"
	"testing"
)

func TestNormalizeStrictness(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, DefaultJudgeStrictness}, // unset maps to default
		{-3, 1},                     // below range clamps to 1
		{1, 1},
		{3, 3},
		{5, 5},
		{9, 5}, // above range clamps to 5
	}
	for _, c := range cases {
		if got := NormalizeStrictness(c.in); got != c.want {
			t.Errorf("NormalizeStrictness(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestStrictnessGuidanceVariesByLevel(t *testing.T) {
	// Each level (after normalisation) must yield a distinct, non-empty instruction
	// so the judge prompt actually changes with the configured strictness.
	seen := make(map[string]int)
	for level := 1; level <= 5; level++ {
		g := strictnessGuidance(level)
		if strings.TrimSpace(g) == "" {
			t.Fatalf("strictnessGuidance(%d) returned empty string", level)
		}
		if prev, ok := seen[g]; ok {
			t.Errorf("strictnessGuidance(%d) duplicates guidance for level %d", level, prev)
		}
		seen[g] = level
	}
}

func TestStrictnessGuidanceEndpoints(t *testing.T) {
	if !strings.Contains(strictnessGuidance(1), "LENIENT") {
		t.Errorf("level 1 guidance should be lenient: %q", strictnessGuidance(1))
	}
	if !strings.Contains(strictnessGuidance(5), "STRICT") {
		t.Errorf("level 5 guidance should be strict: %q", strictnessGuidance(5))
	}
	// The zero value (unset) must fall back to the balanced default.
	if strictnessGuidance(0) != strictnessGuidance(DefaultJudgeStrictness) {
		t.Errorf("unset strictness should match default guidance")
	}
}

func TestTasksFromConfigsNormalisesStrictness(t *testing.T) {
	cfgs := []BenchmarkTaskConfig{
		{TaskID: "a", Category: "coding", ScoringMethod: "llm-judge", JudgeStrictness: 0, Enabled: true},
		{TaskID: "b", Category: "coding", ScoringMethod: "llm-judge", JudgeStrictness: 5, Enabled: true},
		{TaskID: "c", Category: "coding", ScoringMethod: "llm-judge", JudgeStrictness: 42, Enabled: true},
	}
	tasks := TasksFromConfigs(cfgs)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	want := map[string]int{"a": DefaultJudgeStrictness, "b": 5, "c": 5}
	for _, tk := range tasks {
		if tk.JudgeStrictness != want[tk.ID] {
			t.Errorf("task %s: JudgeStrictness = %d, want %d", tk.ID, tk.JudgeStrictness, want[tk.ID])
		}
	}
}
