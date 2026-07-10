package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// The OpenAI Responses API (https://platform.openai.com/docs/api-reference/responses)
// differs from Chat Completions in two ways relevant to PiPiMink:
//
//   - Request: the conversation is passed in "input" (an array of role/content items or
//     a plain string) instead of "messages".
//   - Response: generated text lives in an "output" array. Each item may be a
//     "reasoning" block (skipped) or a "message" block whose "content" array holds
//     "output_text" parts. There is no "choices" array.
//
// Azure AI Foundry serves OpenAI deployments through this API at
// {base}/openai/v1/responses. This file provides the request builder and response
// parser shared by tagging, chat, model selection, and the benchmark judge.

// responsesRequestOptions controls optional fields on a Responses API request.
// Temperature and max_output_tokens are omitted by default because reasoning models
// (o-series, gpt-5.x) reject temperature and count reasoning tokens against the
// output budget, which makes a small fixed budget truncate the answer.
type responsesRequestOptions struct {
	temperature     *float64
	maxOutputTokens int
}

// buildResponsesInput converts an OpenAI-format messages array into the Responses API
// "input" array. Roles are preserved (the Responses API accepts system, developer,
// user, and assistant), and string content is passed through unchanged.
func buildResponsesInput(messages []map[string]interface{}) []map[string]interface{} {
	input := make([]map[string]interface{}, 0, len(messages))
	for _, m := range messages {
		role, _ := m["role"].(string)
		if role == "" {
			continue
		}
		input = append(input, map[string]interface{}{
			"role":    role,
			"content": m["content"],
		})
	}
	return input
}

// buildResponsesPayload assembles a Responses API request body from a messages array.
func buildResponsesPayload(model string, messages []map[string]interface{}, opts responsesRequestOptions) map[string]interface{} {
	payload := map[string]interface{}{
		"model": model,
		"input": buildResponsesInput(messages),
	}
	if opts.temperature != nil {
		payload["temperature"] = *opts.temperature
	}
	if opts.maxOutputTokens > 0 {
		payload["max_output_tokens"] = opts.maxOutputTokens
	}
	return payload
}

// sendResponsesRequest POSTs a Responses API payload and returns the raw response body
// and HTTP status code. Authentication uses a Bearer token, which both the public
// OpenAI Responses API and Azure AI Foundry's /openai/v1 surface accept.
func sendResponsesRequest(url, apiKey string, payload map[string]interface{}, timeout time.Duration) ([]byte, int, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("error marshalling Responses payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, 0, fmt.Errorf("error creating Responses request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error making Responses request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(resp.Body); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("error reading Responses body: %w", err)
	}
	return buf.Bytes(), resp.StatusCode, nil
}

// extractResponsesContent parses the generated text from an OpenAI Responses API body.
//
// It concatenates the text of every "output_text" part inside "message" items, skipping
// "reasoning" and other non-message items. It also honours a top-level "output_text"
// convenience field when present, surfaces API "error" objects, and reports truncation
// via "incomplete_details".
func extractResponsesContent(body []byte) (string, error) {
	var result struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		OutputText        string `json:"output_text"`
		IncompleteDetails *struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error decoding Responses API response: %w", err)
	}

	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}

	var sb strings.Builder
	for _, item := range result.Output {
		if item.Type != "message" {
			continue
		}
		for _, part := range item.Content {
			if part.Type == "output_text" {
				sb.WriteString(part.Text)
			}
		}
	}
	text := sb.String()
	if text == "" && result.OutputText != "" {
		text = result.OutputText
	}

	if text == "" {
		if result.IncompleteDetails != nil && result.IncompleteDetails.Reason == "max_output_tokens" {
			return "", fmt.Errorf("response truncated at max_output_tokens before any text was produced (reasoning consumed the output budget)")
		}
		return "", fmt.Errorf("missing text in Responses API output")
	}
	return text, nil
}
