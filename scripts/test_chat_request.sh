#!/bin/bash

# This script tests the PiPiMink's model selection functionality.
#
# Authentication:
#   Option 1 (Admin API key):  export PIPIMINK_API_KEY=your-admin-api-key
#   Option 2 (Bearer token):   export PIPIMINK_API_KEY=ppm_your-user-bearer-token
#   Option 3 (No auth):        leave unset (works when REQUIRE_AUTH_FOR_CHAT=false, the default)
#
# When REQUIRE_AUTH_FOR_CHAT=true, one of the above must be set.

API_ENDPOINT="http://localhost:8080/chat"
OPENAI_ENDPOINT="http://localhost:8080/v1/chat/completions"

# Build auth header from PIPIMINK_API_KEY env var
AUTH_HEADER=()
if [ -n "${PIPIMINK_API_KEY:-}" ]; then
  if [[ "$PIPIMINK_API_KEY" == ppm_* ]]; then
    # Bearer token (user API token)
    AUTH_HEADER=(-H "Authorization: Bearer $PIPIMINK_API_KEY")
    echo "Using Bearer token authentication"
  else
    # Admin API key
    AUTH_HEADER=(-H "X-API-Key: $PIPIMINK_API_KEY")
    echo "Using X-API-Key authentication"
  fi
else
  echo "No authentication configured (anonymous mode)"
fi

print_response() {
  local response_file="$1"

  if jq . <"$response_file"; then
    return 0
  fi

  echo "Response was not valid JSON. Raw response follows:"
  cat "$response_file"
}

run_native_test() {
  local title="$1"
  local response_file

  echo ""
  echo "=== $title ==="

  response_file=$(mktemp "${TMPDIR:-/tmp}/pipimink-chat-test.XXXXXX")
  if ! curl -sS -X POST \
    -H "Content-Type: application/json" \
    "${AUTH_HEADER[@]}" \
    --data-binary @- \
    "$API_ENDPOINT" >"$response_file"; then
    echo "Request failed"
    rm -f "$response_file"
    return 1
  fi

  print_response "$response_file"
  rm -f "$response_file"
}

run_openai_test() {
  local title="$1"
  local response_file

  echo ""
  echo "=== $title ==="

  response_file=$(mktemp "${TMPDIR:-/tmp}/pipimink-openai-test.XXXXXX")
  if ! curl -sS -X POST \
    -H "Content-Type: application/json" \
    "${AUTH_HEADER[@]}" \
    --data-binary @- \
    "$OPENAI_ENDPOINT" >"$response_file"; then
    echo "Request failed"
    rm -f "$response_file"
    return 1
  fi

  print_response "$response_file"
  rm -f "$response_file"
}

run_native_test "Single-turn: technical question" <<'JSON'
{"message": "Explain quantum computing principles and their application in cryptography, with code examples in Python"}
JSON

run_native_test "Single-turn: creative writing" <<'JSON'
{"message": "Write a short poem about artificial intelligence"}
JSON

run_native_test "Single-turn: mathematical problem" <<'JSON'
{"message": "Solve the differential equation dy/dx = 2xy with the initial condition y(0) = 1"}
JSON

run_native_test "Reasoning: easy deterministic" <<'JSON'
{"message": "A train travels 120 km in 1.5 hours. What is its average speed in km/h? Respond with only the number."}
JSON

run_native_test "Reasoning: medium day-of-week" <<'JSON'
{"message": "Today is Wednesday. A meeting is scheduled 10 days from now. What day of the week is the meeting? Respond with only the day name."}
JSON

run_native_test "Instruction-following: exact output" <<'JSON'
{"message": "Respond with only the single word COMPLETE. No punctuation, no explanation, no other text - just the word COMPLETE."}
JSON

run_native_test "Instruction-following: strict JSON" <<'JSON'
{"message": "Return only a valid JSON object with exactly two keys: \"language\" set to \"Go\" and \"year\" set to 2009. No markdown fences, no explanation."}
JSON

run_native_test "Instruction-following: exact 8-word sentence" <<'JSON'
{"message": "Write a sentence about cats that contains exactly 8 words. Output only the sentence, no explanation."}
JSON

run_native_test "Creative writing: lighthouse story" <<'JSON'
{"message": "Write a short story (150-200 words) about a lighthouse keeper who discovers an unexpected message in a bottle."}
JSON

run_native_test "Summarization: exactly 2 sentences" <<'JSON'
{"message": "Summarize the following passage in exactly 2 sentences: The Industrial Revolution began in Britain in the late 18th century and spread through Europe and North America in the 19th century. Steam power, mechanized production, and factory systems drove urbanization and new social classes. These shifts changed family structures, work conditions, and daily life, and accelerated innovation toward the modern global economy."}
JSON

run_native_test "Factual QA: history" <<'JSON'
{"message": "In what year did World War II end?"}
JSON

run_native_test "Factual QA: science" <<'JSON'
{"message": "What is the chemical symbol for gold?"}
JSON

run_native_test "Go: easy word frequency" <<'JSON'
{"message": "Write a Go function func WordFrequency(text string) map[string]int that returns a map of each word to its occurrence count. Words should be lowercased and split on whitespace. Punctuation attached to words should be stripped. Provide only the function."}
JSON

run_native_test "Go: medium concurrent fan-out" <<'JSON'
{"message": "Write a Go function func FanOut(tasks []func() (string, error), maxWorkers int) ([]string, error) that executes the given tasks concurrently using at most maxWorkers goroutines. Return results in the same order as the input tasks slice. If any task returns an error, return that error. Provide only the function."}
JSON

run_native_test "Go: hard generic B-tree" <<'JSON'
{"message": "Write a Go implementation of a generic B-tree with minimum degree t=2. Define type BTree[K cmp.Ordered, V any] struct with methods Insert, Search, and InOrder. The tree must correctly split nodes on overflow. Use Go 1.21+ generics with the cmp package. Provide the complete implementation."}
JSON

run_native_test "C#: easy reverse words" <<'JSON'
{"message": "Write a C# method public static string ReverseWords(string sentence) that reverses the order of words in a sentence while normalizing whitespace. Multiple spaces become a single space and leading/trailing spaces are removed. Provide only the method."}
JSON

run_native_test "C#: medium generic LRU cache" <<'JSON'
{"message": "Write a C# class LruCache<TKey, TValue> with a fixed capacity. It must support a constructor LruCache(int capacity), TValue Get(TKey key) that marks the item as recently used and throws KeyNotFoundException if missing, and void Put(TKey key, TValue value). All operations must be O(1). Provide only the class definition."}
JSON

run_native_test "C#: hard expression evaluator" <<'JSON'
{"message": "Write a C# class ExpressionEvaluator with a method public static double Evaluate(string expression) that parses and evaluates mathematical expressions with +, -, *, /, parentheses, unary minus, and correct operator precedence. Throw FormatException for malformed input and DivideByZeroException for division by zero. Provide only the class."}
JSON

run_native_test "Python: easy flatten nested list" <<'JSON'
{"message": "Write a Python function def flatten(nested: list) -> list that recursively flattens an arbitrarily nested list of integers. Do not use external libraries. Provide only the function."}
JSON

run_native_test "Python: medium retry decorator" <<'JSON'
{"message": "Write a Python decorator retry(max_attempts: int = 3, delay: float = 1.0, backoff: float = 2.0, exceptions: tuple = (Exception,)) that retries a function on failure with exponential backoff. It must preserve the original function name and docstring and work with *args/**kwargs. Provide only the decorator."}
JSON

run_native_test "Python: hard metaclass mini ORM" <<'JSON'
{"message": "Write a Python mini-ORM using metaclasses. Define field descriptors StringField, IntField, BoolField and a Model base class with a ModelMeta metaclass. Models should auto-detect fields, validate assignment, provide to_dict(), provide from_dict(cls, data), and generate __table_name__ from the class name. Provide the complete implementation."}
JSON

run_native_test "React: easy component with props" <<'JSON'
{"message": "Write a React component UserBadge in TypeScript that accepts props { name: string; role: string; online: boolean } and renders a compact badge. Use a functional component, no class components, and provide only the component code."}
JSON

run_native_test "React: medium todo list with state and filters" <<'JSON'
{"message": "Write a React TodoList component in TypeScript that supports adding todos, toggling completion, and filtering by all, active, and completed. Use hooks, keep state local, and provide only the component code."}
JSON

run_native_test "React: hard data-fetching dashboard" <<'JSON'
{"message": "Write a React dashboard component in TypeScript that fetches a list of projects from /api/projects, shows loading and error states, supports retry, groups projects by status, and uses modern React patterns. Provide only the component code."}
JSON

run_native_test "TypeScript: easy typed emitter" <<'JSON'
{"message": "Write a TypeScript class TypedEmitter<Events extends Record<string, unknown[]>> with on, off, and emit methods for type-safe events. Include a usage example with message: [string, number] and close: []. Provide the class and example."}
JSON

run_native_test "TypeScript: medium schema validator" <<'JSON'
{"message": "Write a TypeScript runtime schema validator inspired by Zod. Define S.string(), S.number(), S.boolean(), S.object({...}), S.array(schema), and S.optional(schema). Each schema must expose parse(input) and safeParse(input). The output type must be inferred automatically. Provide the implementation."}
JSON

run_native_test "TypeScript: hard SQL query builder" <<'JSON'
{"message": "Write a TypeScript type-safe SQL query builder that prevents invalid queries at compile time. Support select, where, join, orderBy, and build. The builder must enforce that select() is called before where() and that value types match column types. Provide the complete implementation."}
JSON

run_native_test "Java: easy stack from queues" <<'JSON'
{"message": "Write a Java generic class QueueStack<T> that implements a stack using only two java.util.LinkedList instances as queues. Implement push, pop, peek, isEmpty, and size. push should be O(1) and pop should be O(n). Provide only the class."}
JSON

run_native_test "Java: medium stream pipeline" <<'JSON'
{"message": "Write a Java class TransactionAnalyzer with a record Transaction(String customer, String category, double amount, LocalDate date) and static methods totalByCustomer, largestInCategory, and topNByCategory using Java Streams. Use Java 17+ features. Provide only the class."}
JSON

run_native_test "Java: hard lock-free queue" <<'JSON'
{"message": "Write a Java generic lock-free concurrent queue class LockFreeQueue<T> using AtomicReference and compare-and-swap. Implement enqueue, dequeue, and isEmpty using the Michael-Scott algorithm with a sentinel node. Provide only the class."}
JSON

run_native_test "Security: Go path traversal review" <<'JSON'
{"message": "Review this Go HTTP handler for security vulnerabilities. Identify all issues, explain exploitation paths, and provide a secure version: func downloadHandler(w http.ResponseWriter, r *http.Request) { filename := r.URL.Query().Get(\"file\"); filepath := \"./uploads/\" + filename; data, err := os.ReadFile(filepath); if err != nil { http.Error(w, fmt.Sprintf(\"Error reading %s: %v\", filepath, err), 500); return }; w.Header().Set(\"Content-Type\", \"application/octet-stream\"); w.Header().Set(\"Content-Disposition\", fmt.Sprintf(\"attachment; filename=%s\", filename)); w.Write(data) }"}
JSON

run_native_test "Security: Python Flask SQL injection and XSS review" <<'JSON'
{"message": "Review the following Python Flask endpoint for security vulnerabilities. Identify all vulnerabilities, explain why each is dangerous, and provide a corrected version: @app.route('/search') def search(): query = request.args.get('q'); sql = f\"SELECT * FROM products WHERE name LIKE '%{query}%'\"; results = db.engine.execute(sql); return render_template_string(f'<h1>Results for {query}</h1>' + ''.join(f'<p>{row.name}: ${row.price}</p>' for row in results))"}
JSON

run_native_test "Security: Node.js JWT authentication review" <<'JSON'
{"message": "Review this Node.js Express authentication middleware for security vulnerabilities. Identify all issues, explain the attack vectors, and provide a secure implementation: const jwt = require('jsonwebtoken'); function authMiddleware(req, res, next) { const token = req.headers.authorization; try { const decoded = jwt.verify(token, 'mysecretkey123'); req.user = decoded; next(); } catch(e) { res.status(401).json({ error: e.message }); } } app.post('/login', (req, res) => { const { username, password } = req.body; const user = users.find(u => u.username === username && u.password === password); if (user) { const token = jwt.sign({ id: user.id, role: user.role, password: user.password }, 'mysecretkey123'); res.json({ token }); } else { res.status(401).json({ error: 'Invalid credentials' }); } });"}
JSON

run_native_test "Multi-turn: conversation history (native /chat)" <<'JSON'
{
  "messages": [
    {"role": "user", "content": "How do I implement a binary search tree in Go?"},
    {"role": "assistant", "content": "A binary search tree in Go can be implemented with a Node struct containing a value and left/right pointers..."},
    {"role": "user", "content": "Can you now show me how to add a delete operation to that implementation?"}
  ]
}
JSON

run_openai_test "Multi-turn: conversation history (OpenAI-compatible /v1/chat/completions)" <<'JSON'
{
  "model": "gpt-4-turbo",
  "messages": [
    {"role": "user", "content": "What is the difference between a mutex and a semaphore?"},
    {"role": "assistant", "content": "A mutex provides mutual exclusion for a single resource, while a semaphore controls access to a pool of resources..."},
    {"role": "user", "content": "Give me a concrete Go example showing both."}
  ]
}
JSON
