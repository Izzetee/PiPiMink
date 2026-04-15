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
	CategoryCodingSecurity       Category = "coding-security"
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
		CategoryCodingSecurity,
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
		// ── C# ──
		{
			ID:            "coding-csharp-easy-string-reverse",
			Category:      CategoryCoding,
			Prompt:        "Write a C# method `public static string ReverseWords(string sentence)` that reverses the order of words in a sentence while preserving whitespace normalization (multiple spaces become single spaces, leading/trailing spaces removed). For example, \"  hello   world  \" should return \"world hello\". Provide only the method.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Correctly reverses word order; 'hello world' → 'world hello'; handles single-word input by returning it unchanged"},
				{Name: "Language idioms", Description: "Uses idiomatic C# — string.Split with StringSplitOptions.RemoveEmptyEntries, string.Join, or LINQ Reverse(); proper use of the string type"},
				{Name: "Edge cases", Description: "Handles empty string (returns empty), null input (returns empty or throws ArgumentNullException), all-whitespace input, single word"},
				{Name: "Code quality", Description: "Clean C# style: PascalCase method name, readable logic, no unnecessary allocations or complexity"},
			},
		},
		{
			ID:            "coding-csharp-medium-generic-cache",
			Category:      CategoryCoding,
			Prompt:        "Write a C# class `LruCache<TKey, TValue>` that implements a least-recently-used cache with a fixed capacity. It must support: `LruCache(int capacity)` constructor, `TValue Get(TKey key)` that returns the value and marks it as recently used (throws KeyNotFoundException if missing), and `void Put(TKey key, TValue value)` that inserts or updates a key-value pair and evicts the least-recently-used item if at capacity. All operations must be O(1). Provide only the class definition.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Get returns correct values and updates recency; Put inserts new entries, updates existing ones, and evicts the LRU item when at capacity; O(1) time for both operations"},
				{Name: "Language idioms", Description: "Uses idiomatic C# generics with proper constraints; leverages LinkedList<T> and Dictionary<TKey, LinkedListNode<T>> or similar .NET collections"},
				{Name: "Edge cases", Description: "Handles capacity of 1 correctly; updating an existing key does not increase size; Get on missing key throws KeyNotFoundException"},
				{Name: "Code quality", Description: "Well-structured C# class with proper access modifiers, generic type parameter naming (TKey/TValue), and clean separation of concerns"},
			},
		},
		{
			ID:            "coding-csharp-hard-expression-parser",
			Category:      CategoryCoding,
			Prompt:        `Write a C# class ExpressionEvaluator with a method public static double Evaluate(string expression) that parses and evaluates a mathematical expression string supporting: +, -, *, / operators, parentheses for grouping, unary minus (e.g., "-3" or "(-5+2)"), and correct operator precedence (* and / before + and -). Examples: "2+3*4" → 14, "(2+3)*4" → 20, "-3+5" → 2, "10/(2+3)" → 2. Throw FormatException for malformed input and DivideByZeroException for division by zero. Provide only the class.`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Correctly evaluates expressions respecting operator precedence and parentheses; handles nested parentheses like '((2+3)*4)/2' → 10; unary minus works in all positions"},
				{Name: "Language idioms", Description: "Uses idiomatic C# — proper exception types (FormatException, DivideByZeroException), Stack<T>, ReadOnlySpan<char> or string indexing; clean recursive descent or shunting-yard implementation"},
				{Name: "Edge cases", Description: "Handles deeply nested parentheses, consecutive operators as errors, empty input, whitespace between tokens, division by zero, expressions like '--3' as error"},
				{Name: "Code quality", Description: "Clean class structure; static method as specified; well-separated parsing and evaluation logic; no use of eval or runtime compilation"},
			},
		},

		// ── Go ──
		{
			ID:            "coding-go-easy-word-frequency",
			Category:      CategoryCoding,
			Prompt:        "Write a Go function `func WordFrequency(text string) map[string]int` that returns a map of each word to its occurrence count. Words should be lowercased and split on whitespace. Punctuation attached to words (commas, periods, exclamation marks, question marks) should be stripped. For example, WordFrequency(\"Hello, hello world!\") should return map[string]int{\"hello\": 2, \"world\": 1}. Provide only the function.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Correctly counts word frequencies; lowercases all words; strips trailing/leading punctuation from each word; 'Hello, hello' → hello:2"},
				{Name: "Language idioms", Description: "Uses idiomatic Go — strings.Fields for splitting, strings.ToLower, strings.Trim or strings.TrimFunc for punctuation; returns map[string]int not a custom type"},
				{Name: "Edge cases", Description: "Handles empty string (returns empty map), all-punctuation tokens (excluded from map), multiple consecutive spaces, mixed case"},
				{Name: "Code quality", Description: "Clean Go style: short variable names, no unnecessary error returns for a pure function, efficient single-pass implementation"},
			},
		},
		{
			ID:            "coding-go-medium-concurrent-fanout",
			Category:      CategoryCoding,
			Prompt:        "Write a Go function `func FanOut(tasks []func() (string, error), maxWorkers int) ([]string, error)` that executes the given tasks concurrently using at most maxWorkers goroutines. It should return results in the same order as the input tasks slice. If any task returns an error, FanOut should return that error (cancelling remaining work is optional but appreciated). Provide only the function.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Results are returned in input order despite concurrent execution; errors from any task are properly propagated; maxWorkers limit is respected"},
				{Name: "Language idioms", Description: "Uses idiomatic Go concurrency — goroutines with a semaphore channel or errgroup; sync.WaitGroup if needed; no data races; proper channel usage"},
				{Name: "Edge cases", Description: "Handles empty tasks slice (returns nil/empty), maxWorkers of 1 (sequential execution), maxWorkers greater than len(tasks), tasks that panic (ideally recovered)"},
				{Name: "Code quality", Description: "Clean Go concurrency patterns; no goroutine leaks; proper use of defer for cleanup; readable control flow without over-engineering"},
			},
		},
		{
			ID:            "coding-go-hard-btree",
			Category:      CategoryCoding,
			Prompt:        "Write a Go implementation of a generic B-tree with minimum degree t=2 (2-3-4 tree). Define `type BTree[K cmp.Ordered, V any] struct` with methods: `func (bt *BTree[K,V]) Insert(key K, val V)`, `func (bt *BTree[K,V]) Search(key K) (V, bool)`, and `func (bt *BTree[K,V]) InOrder() []K` that returns all keys in sorted order. The tree must correctly split nodes on overflow. Use Go 1.21+ generics with the cmp package. Provide the complete implementation.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Insert correctly splits full nodes during descent (proactive splitting); Search finds existing keys and returns false for missing ones; InOrder returns all keys sorted ascending"},
				{Name: "Language idioms", Description: "Uses Go generics properly with cmp.Ordered; idiomatic struct embedding or composition for nodes; no interface{} boxing"},
				{Name: "Edge cases", Description: "Handles inserting into an empty tree, duplicate keys (update value), splitting the root node, searching an empty tree"},
				{Name: "Code quality", Description: "Well-structured with separate node and tree types; clear separation of split logic; methods have correct receiver types (pointer receivers); no unnecessary allocations"},
			},
		},

		// ── Rust ──
		{
			ID:            "coding-rust-easy-temperature-converter",
			Category:      CategoryCoding,
			Prompt:        "Write a Rust module with an enum `enum Temperature { Celsius(f64), Fahrenheit(f64), Kelvin(f64) }` and implement methods: `fn to_celsius(&self) -> f64`, `fn to_fahrenheit(&self) -> f64`, and `fn to_kelvin(&self) -> f64`. Also implement `std::fmt::Display` to format as e.g. '100.0°C', '212.0°F', '373.15K'. Provide only the enum definition and impl blocks.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Conversion formulas are accurate: C→F = C*9/5+32, C→K = C+273.15, and inverse conversions; converting a temperature to its own unit returns the original value"},
				{Name: "Language idioms", Description: "Uses idiomatic Rust — enum with tuple variants, match expressions for conversion, proper impl of std::fmt::Display trait with write! macro"},
				{Name: "Edge cases", Description: "Handles negative temperatures correctly; handles 0.0 values; floating point precision is reasonable"},
				{Name: "Code quality", Description: "Clean Rust style: no unnecessary clones or borrows; &self receivers; proper use of f64; well-formatted Display output"},
			},
		},
		{
			ID:            "coding-rust-medium-thread-safe-counter",
			Category:      CategoryCoding,
			Prompt:        "Write a Rust struct `SharedCounter` that can be safely shared across threads. It must support: `fn new(initial: i64) -> Self`, `fn increment(&self) -> i64` (returns new value), `fn decrement(&self) -> i64` (returns new value), `fn get(&self) -> i64`, and `fn reset(&self, val: i64)`. Demonstrate its use by spawning 10 threads that each increment the counter 1000 times, then assert the final value. Provide the struct, its impl, and the demonstration in a main function.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Counter is truly thread-safe — no data races; final value after 10 threads × 1000 increments equals initial + 10000; all methods return/set correct values"},
				{Name: "Language idioms", Description: "Uses idiomatic Rust concurrency — Arc<AtomicI64> with Ordering::SeqCst or Relaxed, or Arc<Mutex<i64>>; uses std::thread::spawn with move closures; JoinHandle collection and joining"},
				{Name: "Edge cases", Description: "Handles concurrent increment and decrement simultaneously; reset while other threads are running is safe; get returns consistent snapshot"},
				{Name: "Code quality", Description: "Clean Rust ownership model usage; no unnecessary unsafe; proper use of Arc::clone; demonstration code is clear and idiomatic"},
			},
		},
		{
			ID:            "coding-rust-hard-async-rate-limiter",
			Category:      CategoryCoding,
			Prompt:        `Write a Rust async rate limiter using tokio. Define struct RateLimiter implementing a token bucket algorithm with: async fn new(rate: f64, burst: usize) -> Self where rate is tokens per second and burst is max tokens, async fn acquire(&self) that waits until a token is available, and fn try_acquire(&self) -> bool that returns immediately. The limiter must be safe to share across tokio tasks (Send + Sync). Provide the struct, impl, and a brief tokio::main example showing 5 tasks sharing one limiter. Use only tokio and std — no external rate-limiting crates.`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Token bucket algorithm is correctly implemented: tokens replenish at the specified rate up to burst capacity; acquire blocks when no tokens available; try_acquire is non-blocking"},
				{Name: "Language idioms", Description: "Uses idiomatic async Rust — tokio::sync::Mutex or atomic operations; Arc for sharing across tasks; tokio::time::sleep for waiting; proper async/await patterns"},
				{Name: "Edge cases", Description: "Handles burst=0 (always blocks or always fails), rate=0 (no replenishment), multiple concurrent acquires fairly, time calculation precision with Instant"},
				{Name: "Code quality", Description: "Clean async Rust; no unnecessary blocking in async context; proper Send + Sync bounds; tokio::main example compiles and demonstrates the limiter"},
			},
		},

		// ── Java ──
		{
			ID:            "coding-java-easy-stack-from-queues",
			Category:      CategoryCoding,
			Prompt:        "Write a Java generic class `public class QueueStack<T>` that implements a stack (LIFO) using only two java.util.LinkedList instances as queues. Implement: `void push(T item)`, `T pop()` (throws NoSuchElementException if empty), `T peek()` (throws NoSuchElementException if empty), `boolean isEmpty()`, and `int size()`. The push operation should be O(1) and pop should be O(n). Provide only the class.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "LIFO ordering is maintained: push(1), push(2), push(3), pop() → 3, pop() → 2; peek returns top without removing; size tracks correctly"},
				{Name: "Language idioms", Description: "Uses Java generics properly; LinkedList used as Queue via offer/poll methods; throws NoSuchElementException from java.util; proper access modifiers (public/private)"},
				{Name: "Edge cases", Description: "Pop and peek on empty stack throw NoSuchElementException; handles single-element stack; size returns 0 after popping all elements"},
				{Name: "Code quality", Description: "Clean Java style: proper class structure, meaningful field names, correct generic type parameter, no raw types or unchecked casts"},
			},
		},
		{
			ID:            "coding-java-medium-stream-pipeline",
			Category:      CategoryCoding,
			Prompt:        `Write a Java class TransactionAnalyzer with a record record Transaction(String customer, String category, double amount, LocalDate date) and the following static methods using Java Streams: (1) Map<String, Double> totalByCustomer(List<Transaction> txns) — total amount per customer, (2) Optional<Transaction> largestInCategory(List<Transaction> txns, String category) — highest amount transaction in a category, (3) Map<String, List<Transaction>> topNByCategory(List<Transaction> txns, int n) — top n transactions by amount per category. Use Java 17+ features. Provide only the class.`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "totalByCustomer sums correctly across duplicate customers; largestInCategory returns Optional.empty for unknown category; topNByCategory returns at most n items per category sorted by amount descending"},
				{Name: "Language idioms", Description: "Uses Java Streams idiomatically — Collectors.groupingBy, Collectors.summingDouble, Stream.sorted with Comparator; uses record syntax; Optional correctly"},
				{Name: "Edge cases", Description: "Handles empty transaction list for all methods; largestInCategory with no matching transactions; topNByCategory with n larger than group size; null-safe"},
				{Name: "Code quality", Description: "Clean modern Java: record for data, static methods, proper imports implied, no mutable state, readable stream pipelines without excessive nesting"},
			},
		},
		{
			ID:            "coding-java-hard-lock-free-queue",
			Category:      CategoryCoding,
			Prompt:        `Write a Java generic lock-free concurrent queue public class LockFreeQueue<T> using AtomicReference and CAS operations (compare-and-swap). Implement: void enqueue(T item), T dequeue() (returns null if empty), boolean isEmpty(). Use the Michael-Scott algorithm with a sentinel node. The queue must be safe for multiple producer and multiple consumer threads without any locks or synchronized blocks. Provide only the class.`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Michael-Scott algorithm is correctly implemented: enqueue appends to tail with CAS on next pointer, dequeue advances head with CAS; sentinel node separates head and tail; linearizable"},
				{Name: "Language idioms", Description: "Uses java.util.concurrent.atomic.AtomicReference correctly; CAS loops with compareAndSet; no locks, no synchronized; proper use of generics with inner Node class"},
				{Name: "Edge cases", Description: "Handles concurrent enqueue/dequeue from multiple threads; dequeue on empty queue returns null; handles the case where tail falls behind (helping mechanism)"},
				{Name: "Code quality", Description: "Clean implementation with private inner Node class; proper volatile semantics via AtomicReference; no ABA problem (Java GC handles this); well-documented CAS retry loops"},
			},
		},

		// ── TypeScript ──
		{
			ID:            "coding-typescript-easy-type-safe-emitter",
			Category:      CategoryCoding,
			Prompt:        `Write a TypeScript class TypedEmitter<Events extends Record<string, unknown[]>> that provides type-safe event emitting. Implement: on<K extends keyof Events>(event: K, listener: (...args: Events[K]) => void): void, off<K extends keyof Events>(event: K, listener: (...args: Events[K]) => void): void, and emit<K extends keyof Events>(event: K, ...args: Events[K]): void. Include a usage example showing: type MyEvents = { message: [string, number]; close: [] }; const emitter = new TypedEmitter<MyEvents>(). Provide the class and example.`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Events are fully type-safe: emitting 'message' requires (string, number) args; 'close' requires no args; listeners receive correctly typed parameters; on/off/emit work correctly at runtime"},
				{Name: "Language idioms", Description: "Uses advanced TypeScript generics — mapped types, keyof, conditional types or indexed access types; extends Record constraint; proper use of ...args spread typing"},
				{Name: "Edge cases", Description: "off removes only the specified listener, not all listeners for that event; emit with no listeners does nothing; calling off with an unregistered listener is a no-op"},
				{Name: "Code quality", Description: "Clean TypeScript: proper generic constraints, no 'any' type, no type assertions (as); the example demonstrates compile-time type checking"},
			},
		},
		{
			ID:            "coding-typescript-medium-schema-validator",
			Category:      CategoryCoding,
			Prompt:        `Write a TypeScript runtime schema validator inspired by Zod. Define a builder API: S.string(), S.number(), S.boolean(), S.object({...}), S.array(schema), and S.optional(schema). Each schema must have .parse(input: unknown): T that returns the typed value or throws, and .safeParse(input: unknown): { success: true, data: T } | { success: false, error: string }. The type T must be inferred from the schema — no manual type annotation. For example: const userSchema = S.object({ name: S.string(), age: S.number() }); type User = Infer<typeof userSchema> should yield { name: string; age: number }. Provide the implementation.`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "All primitive validators correctly check typeof; object validator checks all keys recursively; array validator checks each element; optional allows undefined; parse throws on invalid, safeParse returns discriminated union"},
				{Name: "Language idioms", Description: "Uses advanced TypeScript type inference — infer keyword, conditional types, mapped types for object schemas; Infer<T> utility type correctly extracts the output type from any schema"},
				{Name: "Edge cases", Description: "Handles nested objects, arrays of objects, optional fields in objects, null vs undefined distinction, extra keys in object input (stripped or allowed)"},
				{Name: "Code quality", Description: "Clean builder pattern; no 'any' in public API types; type inference works without manual annotations; error messages are descriptive"},
			},
		},
		{
			ID:            "coding-typescript-hard-sql-query-builder",
			Category:      CategoryCoding,
			Prompt:        `Write a TypeScript type-safe SQL query builder that prevents invalid queries at compile time. Define a Table type that describes table schemas, then implement a builder with methods: .select(...columns) (only allows columns that exist in the table), .where(column, op, value) (value type must match the column's type), .join(table, on), .orderBy(column, direction), and .build() returning the SQL string. The builder must enforce that select() is called before where(), and build() is called last. Use template literal types for the SQL output type. Example: const q = db.table<UserTable>().select('name','age').where('age','>',18).build(). Provide the complete implementation.`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Select only allows valid column names from the table schema; where enforces value type matches column type; builder produces valid SQL strings; method chaining works correctly"},
				{Name: "Language idioms", Description: "Uses advanced TypeScript features — template literal types, conditional types, phantom types for builder state machine, keyof/Extract for column names; no runtime type assertions needed"},
				{Name: "Edge cases", Description: "Compile error when selecting nonexistent columns; compile error when comparing age (number) with a string value; handles multiple where clauses; join introduces columns from joined table"},
				{Name: "Code quality", Description: "Demonstrates mastery of TypeScript's type system; builder pattern enforces correct method ordering via type narrowing; generated SQL is valid and parameterized"},
			},
		},

		// ── Python ──
		{
			ID:            "coding-python-easy-flatten-nested",
			Category:      CategoryCoding,
			Prompt:        "Write a Python function `def flatten(nested: list) -> list` that recursively flattens an arbitrarily nested list of integers. For example, `flatten([1, [2, [3, 4], 5], [6]])` should return `[1, 2, 3, 4, 5, 6]`. The function must handle any depth of nesting. Do not use any external libraries. Provide only the function.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Correctly flattens deeply nested lists; [1, [2, [3, [4]]]] → [1, 2, 3, 4]; preserves element order; returns integers not nested structures"},
				{Name: "Language idioms", Description: "Uses idiomatic Python — isinstance check for list, recursion or itertools.chain, generator with yield from, or list comprehension; clean Pythonic style"},
				{Name: "Edge cases", Description: "Handles empty list (returns []), list with no nesting (returns copy), deeply nested single element [[[[1]]]] → [1], empty nested lists [[], [1, []]] → [1]"},
				{Name: "Code quality", Description: "Clean, readable Python; proper type hint as specified; no unnecessary imports; efficient without building excessive intermediate lists"},
			},
		},
		{
			ID:            "coding-python-medium-decorator-retry",
			Category:      CategoryCoding,
			Prompt:        `Write a Python decorator retry(max_attempts: int = 3, delay: float = 1.0, backoff: float = 2.0, exceptions: tuple = (Exception,)) that retries a function on failure. It should: (1) retry up to max_attempts times, (2) wait delay seconds between retries with exponential backoff (delay *= backoff after each retry), (3) only catch the specified exception types, (4) re-raise the last exception if all attempts fail, (5) preserve the original function's name and docstring. The decorator must work with both sync functions and accept *args/**kwargs. Provide only the decorator.`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Retries exactly max_attempts times; exponential backoff is applied correctly (delay, delay*backoff, delay*backoff^2); only catches specified exceptions; re-raises on final failure"},
				{Name: "Language idioms", Description: "Uses idiomatic Python — functools.wraps to preserve metadata, time.sleep for delay, proper use of *args/**kwargs, decorator factory pattern with parentheses"},
				{Name: "Edge cases", Description: "max_attempts=1 means one try with no retries; exceptions=() catches nothing; function succeeds on retry returns correct value; non-matching exception types propagate immediately"},
				{Name: "Code quality", Description: "Clean decorator factory pattern; proper closure variable scoping; no mutable default arguments; well-structured retry loop with clear control flow"},
			},
		},
		{
			ID:            "coding-python-hard-metaclass-orm",
			Category:      CategoryCoding,
			Prompt:        `Write a Python mini-ORM using metaclasses. Define field descriptors StringField(max_length=None), IntField(min=None, max=None), BoolField(), and a Model base class with a ModelMeta metaclass. Models should: (1) auto-detect field definitions from class body, (2) validate types and constraints on attribute assignment, (3) provide to_dict() returning field values, (4) provide classmethod from_dict(cls, data) for construction, (5) generate a __table_name__ from the class name (CamelCase → snake_case). Example: class UserProfile(Model): name = StringField(max_length=50); age = IntField(min=0). Assignment user.age = -1 should raise ValueError. Provide the complete implementation.`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Correctness", Description: "Metaclass collects field definitions; descriptors validate on __set__; StringField enforces max_length; IntField enforces min/max; to_dict and from_dict round-trip correctly; table name generation works"},
				{Name: "Language idioms", Description: "Uses Python metaclass correctly (__new__ or __init_subclass__); descriptor protocol (__get__/__set__/__set_name__); classmethod properly; CamelCase to snake_case via regex"},
				{Name: "Edge cases", Description: "Setting wrong type raises TypeError; constraint violations raise ValueError; from_dict with missing fields uses defaults or raises; None handling for optional fields; inheritance between models"},
				{Name: "Code quality", Description: "Clean separation of metaclass, descriptor, and model concerns; no excessive magic; descriptors use __set_name__ (Python 3.6+); readable and maintainable architecture"},
			},
		},

		// ── Coding Security (LLM judge — identify vulnerabilities in provided code) ──
		{
			ID:       "coding-security-sql-injection",
			Category: CategoryCodingSecurity,
			Prompt: `Review the following Python Flask endpoint for security vulnerabilities. Identify ALL vulnerabilities, explain why each is dangerous, and provide a corrected version.

` + "```python" + `
@app.route('/search')
def search():
    query = request.args.get('q')
    sql = f"SELECT * FROM products WHERE name LIKE '%{query}%'"
    results = db.engine.execute(sql)
    return render_template_string(f'<h1>Results for {query}</h1>' +
        ''.join(f'<p>{row.name}: ${row.price}</p>' for row in results))
` + "```",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Vulnerability identification", Description: "Identifies ALL major vulnerabilities: (1) SQL injection via f-string in query, (2) Cross-site scripting (XSS) via render_template_string with unescaped user input, (3) potential information disclosure through raw SQL error messages"},
				{Name: "Explanation quality", Description: "Explains WHY each vulnerability is dangerous with concrete attack examples — e.g., ' OR 1=1 -- for SQLi, <script>alert('xss')</script> for XSS; not just naming the vulnerability"},
				{Name: "Fix correctness", Description: "Corrected code uses parameterized queries (SQLAlchemy text() with bindparams or ORM query), uses render_template with auto-escaping or Markup.escape(), and adds input validation"},
				{Name: "Fix completeness", Description: "The corrected version is a complete, working replacement — not just snippets; uses secure patterns consistently; does not introduce new vulnerabilities in the fix"},
			},
		},
		{
			ID:       "coding-security-jwt-vulnerabilities",
			Category: CategoryCodingSecurity,
			Prompt: `Review this Node.js Express authentication middleware for security vulnerabilities. Identify ALL issues, explain the attack vectors, and provide a secure implementation.

` + "```javascript" + `
const jwt = require('jsonwebtoken');

function authMiddleware(req, res, next) {
  const token = req.headers.authorization;
  try {
    const decoded = jwt.verify(token, 'mysecretkey123');
    req.user = decoded;
    next();
  } catch(e) {
    res.status(401).json({ error: e.message });
  }
}

app.post('/login', (req, res) => {
  const { username, password } = req.body;
  const user = users.find(u => u.username === username && u.password === password);
  if (user) {
    const token = jwt.sign({ id: user.id, role: user.role, password: user.password }, 'mysecretkey123');
    res.json({ token });
  } else {
    res.status(401).json({ error: 'Invalid credentials' });
  }
});
` + "```",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Vulnerability identification", Description: "Identifies ALL issues: (1) hardcoded secret key, (2) password stored in JWT payload, (3) no token expiration, (4) plain-text password comparison (no hashing), (5) no Bearer prefix handling in auth header, (6) error message leaks JWT internal errors to client, (7) no algorithm restriction (algorithm confusion attack)"},
				{Name: "Explanation quality", Description: "Explains attack vectors concretely: secret key can be brute-forced or leaked from source control; password in JWT is exposed via base64 decode; no expiry means tokens never invalidate"},
				{Name: "Fix correctness", Description: "Secure version uses environment variable for secret, bcrypt for password hashing, sets token expiration, strips Bearer prefix, restricts algorithm to HS256, excludes sensitive data from payload"},
				{Name: "Fix completeness", Description: "Provides complete working middleware and login route; all identified vulnerabilities are addressed in the fix; no new vulnerabilities introduced"},
			},
		},
		{
			ID:       "coding-security-path-traversal",
			Category: CategoryCodingSecurity,
			Prompt: `Review this Go HTTP handler for security vulnerabilities. Identify ALL issues, explain how each could be exploited, and provide a secure version.

` + "```go" + `
func downloadHandler(w http.ResponseWriter, r *http.Request) {
    filename := r.URL.Query().Get("file")
    filepath := "./uploads/" + filename

    data, err := os.ReadFile(filepath)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error reading %s: %v", filepath, err), 500)
        return
    }

    w.Header().Set("Content-Type", "application/octet-stream")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
    w.Write(data)
}
` + "```",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Vulnerability identification", Description: "Identifies ALL issues: (1) path traversal via ../../../etc/passwd in filename, (2) information disclosure in error message (leaks filesystem paths and errors), (3) missing Content-Disposition header quoting (header injection), (4) no file size limit (DoS via large file), (5) no authentication/authorization check"},
				{Name: "Explanation quality", Description: "Explains exploitation concretely: ?file=../../../etc/passwd reads system files; error message reveals server directory structure; unquoted filename in Content-Disposition enables header injection"},
				{Name: "Fix correctness", Description: "Secure version uses filepath.Clean and validates the resolved path stays within uploads directory; sanitizes filename; generic error messages; quotes Content-Disposition value; adds file size check"},
				{Name: "Fix completeness", Description: "Provides complete secure handler; uses filepath.Abs to resolve and compare paths; handles all identified vulnerabilities; includes proper error handling without information leakage"},
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
		{
			ID:            "creative-three-distinct-stories",
			Category:      CategoryCreativeWriting,
			Prompt:        "Write 3 short stories (each 100–150 words). Each story must be in a DIFFERENT genre: one science fiction, one literary realism, and one fairy tale. Each must have a distinct protagonist, setting, and conflict. Label them Story 1, Story 2, and Story 3.",
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Genre distinction", Description: "Each story clearly belongs to its assigned genre: science fiction has speculative/technological elements, literary realism depicts plausible everyday life, fairy tale uses mythical/fantastical conventions"},
				{Name: "Story uniqueness", Description: "The three stories have genuinely different protagonists, settings, conflicts, and themes — they do not share plot structure, character archetypes, or narrative beats; no copy-paste variation"},
				{Name: "Individual quality", Description: "Each story independently has a coherent arc (setup, tension, resolution or meaningful ending); prose is engaging; no story feels like filler to meet the count"},
				{Name: "Format compliance", Description: "Exactly 3 stories labeled as instructed; each is between 100 and 150 words; no preamble or commentary outside the stories"},
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
		{
			ID:       "summarization-hard-dense-technical",
			Category: CategorySummarization,
			Prompt: `Summarize the following passage in exactly 3 bullet points. Each bullet must be one sentence of no more than 20 words. You must capture the central thesis, the methodology, and the key finding. Do not include any information not present in the passage.

Passage:
"Recent advances in transformer architectures have enabled significant progress in protein structure prediction. DeepMind's AlphaFold2 system, released in 2021, demonstrated that a neural network trained on the Protein Data Bank's ~170,000 known structures could predict the 3D structure of proteins with accuracy rivaling experimental methods such as X-ray crystallography and cryo-electron microscopy. The system uses a novel attention-based architecture called Evoformer that processes both multiple sequence alignments (MSAs) and pairwise residue representations simultaneously, enabling it to reason about evolutionary and physical constraints jointly. In the CASP14 competition, AlphaFold2 achieved a median Global Distance Test (GDT) score of 92.4 across all targets, compared to the next-best system's score of 67.0 — a margin unprecedented in the competition's 26-year history. This breakthrough has since accelerated drug discovery pipelines, with pharmaceutical companies reporting 30-50% reductions in early-stage target identification timelines. However, critics note that AlphaFold2 still struggles with intrinsically disordered proteins (approximately 30% of the human proteome) and cannot yet predict the effects of point mutations on protein stability, limiting its utility for certain clinical applications."`,
			ScoringMethod: ScoringLLMJudge,
			JudgeCriteria: []JudgeCriterion{
				{Name: "Central thesis capture", Description: "First bullet accurately conveys that transformer-based AI (AlphaFold2) can predict protein structures at experimental accuracy; does not omit the core claim or dilute it"},
				{Name: "Methodology accuracy", Description: "One bullet mentions the Evoformer architecture processing MSAs and pairwise residue representations, or the training on Protein Data Bank structures; factually precise to the passage"},
				{Name: "Key finding specificity", Description: "One bullet includes a specific quantitative result: GDT score of 92.4, the margin over competitors, or the 30-50% drug discovery acceleration; vague paraphrasing scores poorly"},
				{Name: "Strict format adherence", Description: "Exactly 3 bullet points; each is exactly one sentence of no more than 20 words; no introductory text, no sub-bullets, no additional commentary; every claim is traceable to the passage"},
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
