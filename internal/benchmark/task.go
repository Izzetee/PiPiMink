// Package benchmark provides model capability benchmarking across defined task categories.
// Each task has a fixed prompt and a scoring method (deterministic, format-check, or LLM judge).
// Results are stored per model/category and surfaced to the routing meta-model as objective signals.
package benchmark

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Category represents a benchmark domain.
type Category string

const (
	CategoryCoding               Category = "coding"
	CategoryReasoning            Category = "reasoning"
	CategoryInstructionFollowing Category = "instruction-following"
	CategoryCreativeWriting      Category = "creative-writing"
	CategorySummarization        Category = "summarization"
	CategoryFactualQA            Category = "factual-qa"
)

// AllCategories returns all supported benchmark categories in a stable order.
func AllCategories() []Category {
	return []Category{
		CategoryCoding,
		CategoryReasoning,
		CategoryInstructionFollowing,
		CategoryCreativeWriting,
		CategorySummarization,
		CategoryFactualQA,
	}
}

// ScoringMethod defines how a task response is evaluated.
type ScoringMethod string

const (
	// ScoringDeterministic checks whether the response contains the expected answer string.
	ScoringDeterministic ScoringMethod = "deterministic"
	// ScoringFormat applies a structural validator function to the response.
	ScoringFormat ScoringMethod = "format"
	// ScoringLLMJudge sends the response to a judge model that returns a 1–10 score.
	ScoringLLMJudge ScoringMethod = "llm-judge"
)

// JudgeCriterion is a single evaluation dimension for the LLM judge.
type JudgeCriterion struct {
	Name        string // short label, e.g. "Correctness"
	Description string // what the judge should look for
}

// Task defines a single benchmark task.
type Task struct {
	ID              string
	Category        Category
	Prompt          string
	ScoringMethod   ScoringMethod
	ExpectedAnswer  string               // ScoringDeterministic: response must contain this string (case-insensitive)
	FormatValidator func(string) float64 // ScoringFormat: returns 0.0–1.0
	JudgeCriteria   []JudgeCriterion     // ScoringLLMJudge: each criterion is scored 0–10 independently; final score = average
}

// AllTasks returns the full task registry.
func AllTasks() []Task {
	return registeredTasks
}

// TasksForCategory returns tasks belonging to the given category.
func TasksForCategory(c Category) []Task {
	var out []Task
	for _, t := range registeredTasks {
		if t.Category == c {
			out = append(out, t)
		}
	}
	return out
}

// registeredTasks holds all benchmark task definitions, initialised once at startup.
var registeredTasks = buildTasks()

// BenchmarkTaskConfig is the DB-storable, JSON-serialisable representation of a Task.
// ScoringFormat tasks store only the prompt here; their validator is looked up at runtime
// via builtinFormatValidators keyed by TaskID.
type BenchmarkTaskConfig struct {
	TaskID         string           `json:"task_id"`
	Category       string           `json:"category"`
	Prompt         string           `json:"prompt"`
	ScoringMethod  string           `json:"scoring_method"`
	ExpectedAnswer string           `json:"expected_answer,omitempty"`
	JudgeCriteria  []JudgeCriterion `json:"judge_criteria,omitempty"`
	Enabled        bool             `json:"enabled"`
	IsBuiltin      bool             `json:"is_builtin"`
	UpdatedAt      string           `json:"updated_at,omitempty"`
}

// builtinFormatValidators maps task IDs for ScoringFormat tasks to their Go validator functions.
// These cannot be serialised to DB, so they are always resolved at runtime by task ID.
var builtinFormatValidators = map[string]func(string) float64{}

// init populates builtinFormatValidators from the compiled-in task list.
func init() {
	for _, t := range registeredTasks {
		if t.ScoringMethod == ScoringFormat && t.FormatValidator != nil {
			builtinFormatValidators[t.ID] = t.FormatValidator
		}
	}
}

// DefaultTaskConfigs converts the compiled-in task list to BenchmarkTaskConfig records
// suitable for seeding the database on first run.
func DefaultTaskConfigs() []BenchmarkTaskConfig {
	cfgs := make([]BenchmarkTaskConfig, 0, len(registeredTasks))
	for _, t := range registeredTasks {
		cfgs = append(cfgs, BenchmarkTaskConfig{
			TaskID:         t.ID,
			Category:       string(t.Category),
			Prompt:         t.Prompt,
			ScoringMethod:  string(t.ScoringMethod),
			ExpectedAnswer: t.ExpectedAnswer,
			JudgeCriteria:  t.JudgeCriteria,
			Enabled:        true,
			IsBuiltin:      true,
		})
	}
	return cfgs
}

// TasksFromConfigs reconstructs a []Task from DB-loaded configs.
// ScoringFormat validators are re-attached from builtinFormatValidators by task ID.
// Configs with enabled=false are excluded.
func TasksFromConfigs(configs []BenchmarkTaskConfig) []Task {
	tasks := make([]Task, 0, len(configs))
	for _, c := range configs {
		if !c.Enabled {
			continue
		}
		t := Task{
			ID:             c.TaskID,
			Category:       Category(c.Category),
			Prompt:         c.Prompt,
			ScoringMethod:  ScoringMethod(c.ScoringMethod),
			ExpectedAnswer: c.ExpectedAnswer,
			JudgeCriteria:  c.JudgeCriteria,
		}
		if t.ScoringMethod == ScoringFormat {
			t.FormatValidator = builtinFormatValidators[c.TaskID]
		}
		tasks = append(tasks, t)
	}
	return tasks
}

// DefaultTaggingPrompts returns the default tagging prompt texts as a map keyed by prompt name.
func DefaultTaggingPrompts() map[string]string {
	return map[string]string{
		"tagging_system":     defaultTaggingSystem,
		"tagging_user":       defaultTaggingUser,
		"tagging_user_nosys": defaultTaggingUserNoSys,
	}
}

// Tagging prompt defaults — these mirror the constants in internal/llm/model_tags.go
// and are kept in sync manually. Stored here so the benchmark package can seed them to DB
// without creating a circular import.
const defaultTaggingSystem = `You are a routing capabilities assessor for an LLM gateway. Your output will be used to automatically route user prompts to the most suitable language model. Accurate, specific tags lead to better routing decisions.

Your task: describe THIS model's specific task strengths and limitations using lowercase-hyphenated tags.

RULES (follow exactly):
1. Reply ONLY with a valid JSON object — no explanation, no markdown fences, no preamble.
2. Format: {"strengths":["tag1","tag2"],"weaknesses":["tag1","tag2"]}
3. All tags MUST be lowercase-hyphenated (e.g. "code-generation", "step-by-step-reasoning").
4. Tags represent TASK TYPES a user might request — not abstract capability names.
5. Be SPECIFIC to this model's actual capabilities compared to other LLMs.
6. Provide 5–15 strengths and 3–15 weaknesses. Quality over quantity — no padding.
7. If this model does not generate text (e.g. image generation, embeddings, audio-only), return exactly: {"strengths":[],"weaknesses":["not-a-text-generation-model"]}`

const defaultTaggingUser = `Assess THIS model's specific capabilities for task routing. Return only the JSON object.

Example tags (use similar style): "complex-reasoning", "mathematical-problem-solving", "multi-step-code-generation", "code-debugging", "long-context-analysis", "creative-writing", "factual-qa", "text-summarization", "document-extraction", "multilingual-translation", "instruction-following", "structured-data-analysis", "scientific-research", "real-time-information", "function-calling", "vision-understanding", "latex-math", "sql-query-writing".

What does THIS model specifically excel at compared to other LLMs? What tasks does it handle poorly or refuse?`

const defaultTaggingUserNoSys = `You are a routing capabilities assessor for an LLM gateway. Your output routes user prompts to the best model. Return ONLY a JSON object — no markdown, no explanation.

Format: {"strengths":["tag1","tag2"],"weaknesses":["tag1","tag2"]}
Rules: lowercase-hyphenated tags only; 5–15 strengths; 3–15 weaknesses; tags = task types a user might request; if not a text-generation model return {"strengths":[],"weaknesses":["not-a-text-generation-model"]}.

Example tags: "complex-reasoning", "mathematical-problem-solving", "multi-step-code-generation", "code-debugging", "long-context-analysis", "creative-writing", "factual-qa", "text-summarization", "multilingual-translation", "structured-data-analysis", "real-time-information", "function-calling", "vision-understanding".

Assess THIS model's specific strengths and weaknesses for task routing. What does this model excel at compared to other LLMs? What does it handle poorly?`

func buildTasks() []Task {
	return []Task{

		// ── Coding (LLM judge — we evaluate code quality without executing it) ──
		{
			ID:            "coding-prime-check",
			Category:      CategoryCoding,
			Prompt:        "Write a Python function called `is_prime(n)` that returns True if n is a prime number and False otherwise. The function must correctly handle edge cases: 0, 1, negative numbers, and 2. Provide only the function definition.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Returns correct results for edge cases: 0→False, 1→False, 2→True, negative→False, primes→True, composites→False"},
				{Name: "Efficiency", Description: "Uses an efficient algorithm such as trial division up to sqrt(n), not brute-force O(n)"},
				{Name: "Clarity", Description: "Code is readable, uses idiomatic Python, well-named variables"},
				{Name: "Completeness", Description: "Provides only the function definition as requested, no extra boilerplate or test code"},
			},
		},
		{
			ID:            "coding-fizzbuzz",
			Category:      CategoryCoding,
			Prompt:        "Write a Python function `fizzbuzz(n: int) -> list[str]` that returns a list of strings for 1 to n where multiples of 3 are 'Fizz', multiples of 5 are 'Buzz', and multiples of both are 'FizzBuzz'.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Logic", Description: "Multiples of both 3 and 5 → 'FizzBuzz', multiples of 3 only → 'Fizz', multiples of 5 only → 'Buzz', otherwise the number as string"},
				{Name: "Return type", Description: "Returns a list, does not print; includes all values from 1 to n inclusive"},
				{Name: "Edge cases", Description: "Handles n=0 (empty list) and n=1 correctly"},
				{Name: "Code quality", Description: "Clean, idiomatic Python; correct type annotation; no unnecessary complexity"},
			},
		},
		{
			ID:            "coding-sql-aggregation",
			Category:      CategoryCoding,
			Prompt:        "Write a SQL query against tables `orders(id, customer_id, amount)` and `customers(id, name)` that returns the top 5 customers by total order amount. Columns: customer name and total. Order descending by total.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "JOIN", Description: "Correctly joins orders and customers on customer_id = customers.id"},
				{Name: "Aggregation", Description: "Uses SUM(amount) and GROUP BY to compute per-customer totals"},
				{Name: "Ordering", Description: "Orders results by total amount descending (DESC)"},
				{Name: "Limit", Description: "Returns only the top 5 rows using LIMIT 5 or equivalent (TOP 5)"},
				{Name: "Column selection", Description: "Output includes customer name and total amount with meaningful aliases"},
			},
		},
		{
			ID:       "coding-bug-fix",
			Category: CategoryCoding,
			Prompt: `Fix the bug in this Python function:
` + "```python" + `
def find_max(lst):
    max_val = 0
    for i in range(len(lst)):
        if lst[i] > max_val:
            max_val = lst[i]
    return max_val
` + "```" + `
The function returns wrong results for lists containing only negative numbers. Provide the corrected function.`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Bug identification", Description: "Correctly identifies that initialising max_val=0 fails when all values are negative"},
				{Name: "Fix correctness", Description: "Provides a correct fix: initialise to lst[0], float('-inf'), or use built-in max()"},
				{Name: "Edge cases", Description: "Fixed function handles empty lists or raises an appropriate error; does not silently return 0 for empty input"},
				{Name: "Code clarity", Description: "Solution is clean, readable Python; avoids unnecessary complexity"},
			},
		},
		{
			ID:            "coding-go-reverse-unicode",
			Category:      CategoryCoding,
			Prompt:        "Write a Go function `reverseString(s string) string` that reverses a string while correctly handling Unicode (runes, not bytes).",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Unicode correctness", Description: "Uses []rune conversion rather than []byte, so multi-byte characters are not split"},
				{Name: "Go correctness", Description: "Valid Go syntax, correct function signature `reverseString(s string) string`, compiles cleanly"},
				{Name: "Logic", Description: "Reversal logic is correct: first rune becomes last, last becomes first"},
				{Name: "Edge cases", Description: "Handles empty string (returns empty string) without panicking"},
			},
		},

		// ── Reasoning / Math (deterministic — check for known numeric answer) ──
		{
			ID:             "reasoning-train-speed",
			Category:       CategoryReasoning,
			Prompt:         "A train travels 120 km in 1.5 hours. What is its average speed in km/h? Respond with only the number.",
			ScoringMethod:  ScoringDeterministic,
			ExpectedAnswer: "80",
		},
		{
			ID:             "reasoning-multiplication",
			Category:       CategoryReasoning,
			Prompt:         "What is 23 × 47? Respond with only the number.",
			ScoringMethod:  ScoringDeterministic,
			ExpectedAnswer: "1081",
		},
		{
			ID:             "reasoning-percentage-remaining",
			Category:       CategoryReasoning,
			Prompt:         "A warehouse stores 240 items. 30% are shipped out. How many items remain? Respond with only the number.",
			ScoringMethod:  ScoringDeterministic,
			ExpectedAnswer: "168",
		},
		{
			ID:             "reasoning-rectangle-area",
			Category:       CategoryReasoning,
			Prompt:         "A rectangle has width 8 and height 6. What is its area? Respond with only the number.",
			ScoringMethod:  ScoringDeterministic,
			ExpectedAnswer: "48",
		},
		{
			ID:             "reasoning-days-of-week",
			Category:       CategoryReasoning,
			Prompt:         "Today is Wednesday. A meeting is scheduled 10 days from now. What day of the week is the meeting? Respond with only the day name.",
			ScoringMethod:  ScoringDeterministic,
			ExpectedAnswer: "Saturday",
		},

		// ── Instruction Following (format validators — no LLM judge needed) ────
		{
			ID:            "instruction-exact-word",
			Category:      CategoryInstructionFollowing,
			Prompt:        "Respond with only the single word COMPLETE. No punctuation, no explanation, no other text — just the word COMPLETE.",
			ScoringMethod: ScoringFormat,
			FormatValidator: func(response string) float64 {
				if strings.TrimSpace(response) == "COMPLETE" {
					return 1.0
				}
				return 0.0
			},
		},
		{
			ID:            "instruction-json-output",
			Category:      CategoryInstructionFollowing,
			Prompt:        `Return only a valid JSON object with exactly two keys: "language" set to "Go" and "year" set to 2009. No markdown fences, no explanation.`,
			ScoringMethod: ScoringFormat,
			FormatValidator: func(response string) float64 {
				trimmed := strings.TrimSpace(response)
				// Strip optional markdown fence
				trimmed = strings.TrimPrefix(trimmed, "```json")
				trimmed = strings.TrimPrefix(trimmed, "```")
				trimmed = strings.TrimSuffix(trimmed, "```")
				trimmed = strings.TrimSpace(trimmed)
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
					return 0.0
				}
				score := 0.0
				if lang, ok := obj["language"].(string); ok && lang == "Go" {
					score += 0.5
				}
				switch v := obj["year"].(type) {
				case float64:
					if int(v) == 2009 {
						score += 0.5
					}
				}
				return score
			},
		},
		{
			ID:            "instruction-numbered-list",
			Category:      CategoryInstructionFollowing,
			Prompt:        "List exactly 3 European capital cities. Format as:\n1. [city]\n2. [city]\n3. [city]\nOutput only the list, nothing else.",
			ScoringMethod: ScoringFormat,
			FormatValidator: func(response string) float64 {
				re := regexp.MustCompile(`(?m)^\d+\.\s+\S`)
				matches := re.FindAllString(response, -1)
				switch len(matches) {
				case 3:
					return 1.0
				case 1, 2:
					return 0.5
				default:
					return 0.0
				}
			},
		},
		{
			ID:            "instruction-exact-repeat",
			Category:      CategoryInstructionFollowing,
			Prompt:        "Repeat the following text exactly as written and nothing else: 'The quick brown fox jumps over the lazy dog'",
			ScoringMethod: ScoringFormat,
			FormatValidator: func(response string) float64 {
				if strings.Contains(strings.TrimSpace(response), "The quick brown fox jumps over the lazy dog") {
					return 1.0
				}
				return 0.0
			},
		},
		{
			ID:            "instruction-word-count",
			Category:      CategoryInstructionFollowing,
			Prompt:        "Write a sentence about cats that contains exactly 8 words. Output only the sentence, no explanation.",
			ScoringMethod: ScoringFormat,
			FormatValidator: func(response string) float64 {
				words := strings.Fields(strings.TrimSpace(response))
				if len(words) == 8 {
					return 1.0
				}
				return 0.0
			},
		},

		// ── Creative Writing (LLM judge) ─────────────────────────────────────
		{
			ID:            "creative-lighthouse-story",
			Category:      CategoryCreativeWriting,
			Prompt:        "Write a short story (150–200 words) about a lighthouse keeper who discovers an unexpected message in a bottle.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Narrative coherence", Description: "The story has a clear arc: setup, event, and resolution or emotional close"},
				{Name: "Originality", Description: "The premise or the content of the message is creative and non-generic"},
				{Name: "Prose quality", Description: "Language is vivid, engaging, and appropriate for a short story; avoids clichés"},
				{Name: "Length compliance", Description: "Story is between 150 and 200 words as instructed"},
			},
		},
		{
			ID:            "creative-morning-poem",
			Category:      CategoryCreativeWriting,
			Prompt:        "Write a poem about the feeling of early morning. It must have at least 3 stanzas.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Imagery", Description: "Uses concrete sensory details that evoke the sights, sounds, or feelings of early morning"},
				{Name: "Emotional resonance", Description: "Conveys a mood or feeling authentically, not just describing events mechanically"},
				{Name: "Structure", Description: "Has at least 3 distinct stanzas with coherent internal structure"},
				{Name: "Originality", Description: "Avoids overused morning clichés; brings a fresh perspective or metaphor"},
			},
		},
		{
			ID:            "creative-product-description",
			Category:      CategoryCreativeWriting,
			Prompt:        "Write a 3-sentence product description for a fictional artisan coffee blend called 'Midnight Ember'. Make it feel premium and evocative.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Sensory language", Description: "Uses specific, evocative sensory details (taste, aroma, texture, warmth) rather than generic adjectives"},
				{Name: "Premium tone", Description: "Writing feels luxurious and artisan; suits a high-end product brand voice"},
				{Name: "Format compliance", Description: "Exactly 3 sentences — no more, no less"},
				{Name: "Specificity", Description: "References 'Midnight Ember' by name; avoids generic coffee marketing phrases like 'rich and bold'"},
			},
		},

		// ── Summarization (LLM judge) ─────────────────────────────────────────
		{
			ID:       "summarization-industrial-revolution",
			Category: CategorySummarization,
			Prompt: `Summarize the following passage in exactly 2 sentences:

"The Industrial Revolution, which began in Britain in the late 18th century and spread throughout Europe and North America during the 19th century, fundamentally transformed human society. Before industrialization, most people lived in rural areas and worked in agriculture or cottage industries. The development of steam power, mechanized production, and factory systems led to mass urbanization as workers moved to cities to work in factories. This shift created new social classes — the industrial capitalist class and the urban working class — and led to profound changes in family structure, working conditions, and daily life. The revolution also accelerated technological innovation, laying the groundwork for the modern global economy."`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Key facts coverage", Description: "Mentions the origin in Britain, the spread to Europe/N. America, and at least one major consequence (urbanization, social class changes, or technological innovation)"},
				{Name: "Accuracy", Description: "Contains no hallucinated facts; every claim is supported by the source passage"},
				{Name: "Conciseness", Description: "Information is compressed effectively; no padding or repetition of the source's wording"},
				{Name: "Format compliance", Description: "Summary is exactly 2 sentences as instructed"},
			},
		},
		{
			ID:       "summarization-remote-work",
			Category: CategorySummarization,
			Prompt: `Extract the 3 most important points from the following text as a bulleted list:

"Remote work has become increasingly common since 2020. Studies show that remote workers report higher job satisfaction on average, primarily due to eliminated commute times and greater schedule flexibility. However, remote work also presents challenges: many workers report feelings of isolation and difficulty separating work from personal life. Companies have seen mixed results — some report maintained or improved productivity, while others cite challenges with collaboration and mentorship of junior employees. Most organizations are now adopting hybrid models that combine in-office and remote work days."`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Point selection", Description: "Selects 3 genuinely important and distinct points from the text (not redundant sub-points)"},
				{Name: "Accuracy", Description: "All extracted points are accurate to the source; no hallucination or distortion"},
				{Name: "Format compliance", Description: "Response uses a bulleted list format with exactly 3 bullets"},
				{Name: "Conciseness", Description: "Each bullet is brief and informative; no unnecessary filler words"},
			},
		},

		// ── Factual QA (deterministic — response must contain the known answer) ─
		{
			ID:             "factual-capital-france",
			Category:       CategoryFactualQA,
			Prompt:         "What is the capital of France?",
			ScoringMethod:  ScoringDeterministic,
			ExpectedAnswer: "Paris",
		},
		{
			ID:             "factual-ww2-end-year",
			Category:       CategoryFactualQA,
			Prompt:         "In what year did World War II end?",
			ScoringMethod:  ScoringDeterministic,
			ExpectedAnswer: "1945",
		},
		{
			ID:             "factual-gold-symbol",
			Category:       CategoryFactualQA,
			Prompt:         "What is the chemical symbol for gold?",
			ScoringMethod:  ScoringDeterministic,
			ExpectedAnswer: "Au",
		},
		{
			ID:             "factual-go-creator",
			Category:       CategoryFactualQA,
			Prompt:         "What company created the Go programming language?",
			ScoringMethod:  ScoringDeterministic,
			ExpectedAnswer: "Google",
		},
		{
			ID:             "factual-http-acronym",
			Category:       CategoryFactualQA,
			Prompt:         "What does HTTP stand for?",
			ScoringMethod:  ScoringDeterministic,
			ExpectedAnswer: "HyperText Transfer Protocol",
		},
	}
}
