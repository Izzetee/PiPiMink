package llm

import (
	"encoding/json"
	"log"
	"strings"
)

// Helper function to get map keys for debugging.
// This is useful for error reporting when a field is missing.
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// extractJSON extracts valid JSON from a message string.
// It handles various formats including code blocks, think tags, and direct JSON.
func (c *Client) extractJSON(message string) string {
	// Check for <think> tags
	thinkStart := strings.Index(message, "<think>")
	var jsonStart, jsonEnd int

	if thinkStart >= 0 {
		// There are think tags, look for the actual JSON content
		log.Println("Detected <think> tags in response, attempting to extract JSON")

		// Look for JSON content inside ```json blocks
		jsonBlockStart := strings.Index(message, "```json")
		if jsonBlockStart >= 0 {
			contentStart := jsonBlockStart + len("```json")
			contentEnd := strings.Index(message[contentStart:], "```")
			if contentEnd >= 0 {
				// Extract content between ```json and ```
				jsonCandidate := message[contentStart : contentStart+contentEnd]
				jsonCandidate = strings.TrimSpace(jsonCandidate)
				var tagData json.RawMessage
				if err := json.Unmarshal([]byte(jsonCandidate), &tagData); err == nil {
					return jsonCandidate
				}
			}
		}

		// If no valid JSON in code blocks, search for the first complete JSON object.
		if candidate := firstJSONObject(message); candidate != "" {
			return candidate
		}
		jsonStart = strings.Index(message, "{")
		jsonEnd = strings.LastIndex(message, "}")
	} else {
		// Try to validate the entire message as JSON
		var tagData json.RawMessage
		if err := json.Unmarshal([]byte(message), &tagData); err == nil {
			return message
		}

		// If not valid JSON, prefer the first complete object. Models occasionally
		// emit a correct answer and then leak another chat-template turn.
		if candidate := firstJSONObject(message); candidate != "" {
			log.Println("Successfully extracted first valid JSON object from response")
			return candidate
		}
		jsonStart = strings.Index(message, "{")
		jsonEnd = strings.LastIndex(message, "}")
	}

	// Extract and validate JSON if we found matching braces
	if jsonStart >= 0 && jsonEnd > jsonStart {
		jsonCandidate := message[jsonStart : jsonEnd+1]
		var tagData json.RawMessage
		if err := json.Unmarshal([]byte(jsonCandidate), &tagData); err == nil {
			log.Println("Successfully extracted valid JSON from response")
			return jsonCandidate
		} else {
			log.Println("Found JSON-like content but it's not valid JSON:", err)
		}

		// Common small-model mistake: unclosed array before the closing brace.
		// Try inserting ']' before the last '}' and re-parse.
		repaired := jsonCandidate[:len(jsonCandidate)-1] + "]}"
		if err2 := json.Unmarshal([]byte(repaired), &tagData); err2 == nil {
			log.Println("Repaired JSON by closing unclosed array before '}'")
			return repaired
		}
	}

	// Return empty JSON structure if we couldn't find valid JSON
	log.Println("Failed to extract valid JSON from response, returning empty JSON")
	return "{\"strengths\":[], \"weaknesses\":[]}"
}

// firstJSONObject returns the first syntactically valid JSON object in text. It tracks
// quoted strings and escapes so braces inside tag values do not terminate the object.
func firstJSONObject(text string) string {
	for start := strings.IndexByte(text, '{'); start >= 0; {
		depth := 0
		inString := false
		escaped := false
	scanCandidate:
		for i := start; i < len(text); i++ {
			ch := text[i]
			if inString {
				if escaped {
					escaped = false
				} else if ch == '\\' {
					escaped = true
				} else if ch == '"' {
					inString = false
				}
				continue
			}
			switch ch {
			case '"':
				inString = true
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					candidate := text[start : i+1]
					var value map[string]interface{}
					if json.Unmarshal([]byte(candidate), &value) == nil {
						return candidate
					}
					break scanCandidate
				}
			}
		}
		next := strings.IndexByte(text[start+1:], '{')
		if next < 0 {
			break
		}
		start += next + 1
	}
	return ""
}
