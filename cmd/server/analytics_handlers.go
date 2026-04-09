package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"PiPiMink/internal/database"
	"PiPiMink/internal/models"
)

// parseTimeRange extracts start and end times from query parameters.
// Supports "range" (1h, 24h, 7d, 30d) or explicit "start"+"end" (RFC3339).
// Defaults to 24h if nothing is provided.
func parseTimeRange(r *http.Request) (time.Time, time.Time) {
	now := time.Now()

	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if endStr := r.URL.Query().Get("end"); endStr != "" {
			start, err1 := time.Parse(time.RFC3339, startStr)
			end, err2 := time.Parse(time.RFC3339, endStr)
			if err1 == nil && err2 == nil {
				return start, end
			}
			// Also try date-only format (YYYY-MM-DD) from the custom date picker
			start, err1 = time.Parse("2006-01-02", startStr)
			end, err2 = time.Parse("2006-01-02", endStr)
			if err1 == nil && err2 == nil {
				// End of day for the end date
				return start, end.Add(24*time.Hour - time.Second)
			}
		}
	}

	rangeStr := r.URL.Query().Get("range")
	switch rangeStr {
	case "1h":
		return now.Add(-1 * time.Hour), now
	case "7d":
		return now.Add(-7 * 24 * time.Hour), now
	case "30d":
		return now.Add(-30 * 24 * time.Hour), now
	default: // "24h" or unspecified
		return now.Add(-24 * time.Hour), now
	}
}

// handleAnalyticsSummary returns KPI summary, model usage, and latency data.
// Auth: admin (enforced by middleware). Admins see all data; regular users see only their own.
func (s *Server) handleAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	start, end := parseTimeRange(r)

	// Scope: admin sees all users, regular user sees only own data
	userFilter := getUserID(r)
	if getAuthLevel(r) >= AuthAdmin {
		userFilter = "" // empty = all users
	}

	// Run all queries (sequentially to keep it simple; they're fast on indexed data)
	kpi, err := s.db.GetKpiSummaryFiltered(start, end, userFilter)
	if err != nil {
		log.Printf("Error fetching KPI summary: %v", err)
		http.Error(w, "error fetching analytics", http.StatusInternalServerError)
		return
	}

	modelUsage, err := s.db.GetModelUsageFiltered(start, end, userFilter)
	if err != nil {
		log.Printf("Error fetching model usage: %v", err)
		http.Error(w, "error fetching analytics", http.StatusInternalServerError)
		return
	}
	if modelUsage == nil {
		modelUsage = []database.ModelUsageRow{}
	}

	latencyPerModel, err := s.db.GetLatencyPerModelFiltered(start, end, userFilter)
	if err != nil {
		log.Printf("Error fetching latency per model: %v", err)
		http.Error(w, "error fetching analytics", http.StatusInternalServerError)
		return
	}
	if latencyPerModel == nil {
		latencyPerModel = []database.LatencyPerModelRow{}
	}

	latencyTimeSeries, err := s.db.GetLatencyTimeSeriesFiltered(start, end, userFilter)
	if err != nil {
		log.Printf("Error fetching latency time series: %v", err)
		http.Error(w, "error fetching analytics", http.StatusInternalServerError)
		return
	}
	if latencyTimeSeries == nil {
		latencyTimeSeries = []database.LatencyTimeSeriesRow{}
	}

	latencyPercentiles, err := s.db.GetLatencyPercentilesFiltered(start, end, userFilter)
	if err != nil {
		log.Printf("Error fetching latency percentiles: %v", err)
		http.Error(w, "error fetching analytics", http.StatusInternalServerError)
		return
	}
	if latencyPercentiles == nil {
		latencyPercentiles = []database.LatencyPercentilesRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"kpiSummary":         kpi,
		"modelUsage":         modelUsage,
		"latencyPerModel":    latencyPerModel,
		"latencyTimeSeries":  latencyTimeSeries,
		"latencyPercentiles": latencyPercentiles,
	})
}

// handleRoutingDecisions returns paginated routing decisions.
// Auth: admin (enforced by middleware). Admins see all; regular users see only their own.
func (s *Server) handleRoutingDecisions(w http.ResponseWriter, r *http.Request) {
	start, end := parseTimeRange(r)

	// Scope: admin sees all users, regular user sees only own data
	userFilter := getUserID(r)
	if getAuthLevel(r) >= AuthAdmin {
		userFilter = "" // empty = all users
	}

	page := 1
	pageSize := 10
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	offset := (page - 1) * pageSize
	decisions, total, err := s.db.GetRoutingDecisionsFiltered(start, end, pageSize, offset, userFilter)
	if err != nil {
		log.Printf("Error fetching routing decisions: %v", err)
		http.Error(w, "error fetching routing decisions", http.StatusInternalServerError)
		return
	}
	if decisions == nil {
		decisions = []database.RoutingDecisionRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"decisions": decisions,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
	})
}

// logRoutingDecision persists a routing decision asynchronously.
func (s *Server) logRoutingDecision(result models.RoutingResult, prompt string, provider string, latencyMs int64, status string, userID string) {
	// Build prompt snippet (first 100 chars)
	snippet := prompt
	if len(snippet) > 100 {
		snippet = snippet[:100] + "..."
	}

	tags := result.MatchingTags
	if tags == nil {
		tags = []string{}
	}
	relevance := result.TagRelevance
	if relevance == nil {
		relevance = make(map[string]float64)
	}

	rd := database.RoutingDecisionRow{
		PromptSnippet:    snippet,
		FullPrompt:       prompt,
		AnalyzedTags:     tags,
		TagRelevance:     relevance,
		SelectedModel:    result.ModelName,
		Provider:         provider,
		RoutingReason:    result.Reason,
		EvaluatorModel:   result.EvaluatorModel,
		EvaluationTimeMs: result.EvaluationTimeMs,
		CacheHit:         result.CacheHit,
		LatencyMs:        latencyMs,
		Status:           status,
		UserID:           userID,
	}

	go func() {
		if err := s.db.SaveRoutingDecision(rd); err != nil {
			log.Printf("Error logging routing decision: %v", err)
		}
	}()
}
