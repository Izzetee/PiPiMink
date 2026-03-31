package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"PiPiMink/internal/config"
)

// Scorer evaluates model responses. Deterministic and format-check tasks are scored locally;
// LLM-judge tasks are sent to a configurable judge model.
type Scorer struct {
	judgeProvider config.ProviderConfig // resolved provider config (with per-model overrides applied)
	judgeModel    string
	httpClient    *http.Client
}

// NewScorer creates a scorer backed by the given judge model endpoint.
// The provider config should already have per-model overrides applied via ForModel().
func NewScorer(judgeProvider config.ProviderConfig, judgeModel string) *Scorer {
	return &Scorer{
		judgeProvider: judgeProvider,
		judgeModel:    judgeModel,
		httpClient:    &http.Client{Timeout: judgeProvider.Timeout},
	}
}

// Score evaluates a model's response to a task and returns a normalised score in [0.0, 1.0].
func (s *Scorer) Score(ctx context.Context, task Task, response string) float64 {
	switch task.ScoringMethod {
	case ScoringDeterministic:
		return s.scoreDeterministic(task.ExpectedAnswer, response)
	case ScoringFormat:
		if task.FormatValidator != nil {
			return task.FormatValidator(response)
		}
		return 0.0
	case ScoringLLMJudge:
		return s.scoreLLMJudge(ctx, task, response)
	default:
		log.Printf("benchmark: unknown scoring method %q for task %s", task.ScoringMethod, task.ID)
		return 0.0
	}
}

// scoreDeterministic returns 1.0 when the response contains the expected answer
// (case-insensitive substring match), 0.0 otherwise.
func (s *Scorer) scoreDeterministic(expected, response string) float64 {
	if strings.Contains(strings.ToLower(response), strings.ToLower(expected)) {
		return 1.0
	}
	return 0.0
}

// scoreLLMJudge asks the judge model to rate each criterion independently on a 0–10
// scale. The final score is the average of all criterion scores, normalised to [0.0, 1.0].
// Returns 0.0 on any error.
func (s *Scorer) scoreLLMJudge(ctx context.Context, task Task, response string) float64 {
	if s.judgeProvider.BaseURL == "" || s.judgeModel == "" {
		log.Printf("benchmark: judge not configured, skipping LLM judge for task %s", task.ID)
		return 0.0
	}

	// Build numbered criteria list for the prompt.
	criteriaLines := make([]string, len(task.JudgeCriteria))
	for i, c := range task.JudgeCriteria {
		criteriaLines[i] = fmt.Sprintf("%d. %s: %s", i+1, c.Name, c.Description)
	}
	criteriaText := strings.Join(criteriaLines, "\n")

	// Build the expected JSON keys so the judge knows what to return.
	exampleKeys := make([]string, len(task.JudgeCriteria))
	for i, c := range task.JudgeCriteria {
		exampleKeys[i] = fmt.Sprintf("%q: <0-10>", c.Name)
	}

	systemMsg := fmt.Sprintf(
		"You are an objective AI response evaluator. Score each criterion independently on a scale of 0 to 10.\n"+
			"Use the full range: 0=completely missing or wrong, 3=poor, 5=mediocre, 7=good, 9=excellent, 10=perfect.\n"+
			"Return ONLY a JSON object in this exact format (no other text):\n"+
			"{\"scores\": {%s}, \"reason\": \"<one sentence overall summary>\"}",
		strings.Join(exampleKeys, ", "),
	)

	userMsg := fmt.Sprintf(
		"Task prompt:\n%s\n\nModel response:\n%s\n\nCriteria to score (0–10 each):\n%s",
		task.Prompt, response, criteriaText,
	)

	var body []byte
	var url string
	var err error

	switch s.judgeProvider.Type {
	case config.ProviderTypeAnthropic:
		body, url, err = s.buildAnthropicRequest(systemMsg, userMsg)
	default:
		body, url, err = s.buildOpenAIRequest(systemMsg, userMsg)
	}
	if err != nil {
		log.Printf("benchmark: judge marshal error for task %s: %v", task.ID, err)
		return 0.0
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("benchmark: judge request creation error for task %s: %v", task.ID, err)
		return 0.0
	}
	req.Header.Set("Content-Type", "application/json")

	switch s.judgeProvider.Type {
	case config.ProviderTypeAnthropic:
		if s.judgeProvider.APIKey != "" {
			req.Header.Set("x-api-key", s.judgeProvider.APIKey)
		}
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		if s.judgeProvider.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+s.judgeProvider.APIKey)
		}
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("benchmark: judge HTTP error for task %s: %v", task.ID, err)
		return 0.0
	}
	defer func() { _ = resp.Body.Close() }()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		log.Printf("benchmark: judge response read error for task %s: %v", task.ID, err)
		return 0.0
	}

	log.Printf("benchmark: judge response status=%d body=%s", resp.StatusCode, buf.String())

	// Extract text content from the response based on provider type.
	content, err := s.extractJudgeContent(buf.Bytes())
	if err != nil {
		log.Printf("benchmark: judge response parse error for task %s: %v", task.ID, err)
		return 0.0
	}

	// Extract the JSON object from the content (the judge may add surrounding text).
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end <= start {
		log.Printf("benchmark: judge returned no JSON for task %s: %s", task.ID, content)
		return 0.0
	}

	var judgeResult struct {
		Scores map[string]float64 `json:"scores"`
		Reason string             `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &judgeResult); err != nil {
		log.Printf("benchmark: judge JSON parse error for task %s: %v (content: %s)", task.ID, err, content)
		return 0.0
	}

	if len(judgeResult.Scores) == 0 {
		log.Printf("benchmark: judge returned no criterion scores for task %s", task.ID)
		return 0.0
	}

	// Average all criterion scores and normalise from [0,10] to [0.0,1.0].
	sum := 0.0
	for name, score := range judgeResult.Scores {
		if score < 0 || score > 10 {
			log.Printf("benchmark: judge out-of-range score %.1f for criterion %q task %s", score, name, task.ID)
			score = max(0, min(10, score))
		}
		sum += score
	}
	normalised := (sum / float64(len(judgeResult.Scores))) / 10.0
	log.Printf("benchmark: judge scores %v (avg=%.2f) for task %s — %s", judgeResult.Scores, normalised, task.ID, judgeResult.Reason)
	return normalised
}

// buildOpenAIRequest builds the JSON payload and URL for an OpenAI-compatible judge request.
func (s *Scorer) buildOpenAIRequest(systemMsg, userMsg string) ([]byte, string, error) {
	payload := map[string]interface{}{
		"model":       s.judgeModel,
		"temperature": 0.0,
		"messages": []map[string]string{
			{"role": "system", "content": systemMsg},
			{"role": "user", "content": userMsg},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	return body, s.judgeProvider.ChatCompletionsURL(), nil
}

// buildAnthropicRequest builds the JSON payload and URL for an Anthropic Messages API judge request.
func (s *Scorer) buildAnthropicRequest(systemMsg, userMsg string) ([]byte, string, error) {
	payload := map[string]interface{}{
		"model":      s.judgeModel,
		"max_tokens": 4096,
		"system":     systemMsg,
		"messages": []map[string]string{
			{"role": "user", "content": userMsg},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	return body, s.judgeProvider.BaseURL + "/v1/messages", nil
}

// extractJudgeContent extracts the text content from a judge API response,
// handling both OpenAI and Anthropic response formats.
func (s *Scorer) extractJudgeContent(body []byte) (string, error) {
	switch s.judgeProvider.Type {
	case config.ProviderTypeAnthropic:
		return extractAnthropicContent(body)
	default:
		return extractOpenAIContent(body)
	}
}

// extractOpenAIContent extracts text from an OpenAI-compatible chat completions response.
func extractOpenAIContent(body []byte) (string, error) {
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("error decoding OpenAI response: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("missing or empty choices in OpenAI response")
	}
	return apiResp.Choices[0].Message.Content, nil
}

// extractAnthropicContent extracts text from an Anthropic Messages API response.
func extractAnthropicContent(body []byte) (string, error) {
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error decoding Anthropic response: %w", err)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("missing or empty content in Anthropic response")
	}
	return result.Content[0].Text, nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
