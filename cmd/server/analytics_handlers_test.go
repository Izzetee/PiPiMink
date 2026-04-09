package server

import (
	"encoding/json"
	"net/http"

	"PiPiMink/internal/database"

	"github.com/stretchr/testify/mock"
)

func (s *ServerTestSuite) TestHandleAnalyticsSummary() {
	// Admin auth via X-API-Key → userFilter="" (all users)
	s.mockDB.On("GetKpiSummaryFiltered", mock.Anything, mock.Anything, "").Return(database.KpiSummary{
		TotalRequests: 100, AvgLatencyMs: 250, MostUsedModel: "gpt-4-turbo", ErrorRate: 0.02,
	}, nil)
	s.mockDB.On("GetModelUsageFiltered", mock.Anything, mock.Anything, "").Return([]database.ModelUsageRow{}, nil)
	s.mockDB.On("GetLatencyPerModelFiltered", mock.Anything, mock.Anything, "").Return([]database.LatencyPerModelRow{}, nil)
	s.mockDB.On("GetLatencyTimeSeriesFiltered", mock.Anything, mock.Anything, "").Return([]database.LatencyTimeSeriesRow{}, nil)
	s.mockDB.On("GetLatencyPercentilesFiltered", mock.Anything, mock.Anything, "").Return([]database.LatencyPercentilesRow{}, nil)

	req, _ := http.NewRequest("GET", "/admin/analytics/summary?range=24h", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)

	var resp map[string]interface{}
	s.Require().NoError(json.Unmarshal(s.recorder.Body.Bytes(), &resp))
	s.Contains(resp, "kpiSummary")
	s.Contains(resp, "modelUsage")
	s.Contains(resp, "latencyPerModel")
	s.Contains(resp, "latencyTimeSeries")
	s.Contains(resp, "latencyPercentiles")
}

func (s *ServerTestSuite) TestHandleAnalyticsSummary_Unauthorized() {
	req, _ := http.NewRequest("GET", "/admin/analytics/summary", nil)
	// No API key
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusUnauthorized, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleAnalyticsSummary_DBError() {
	s.mockDB.On("GetKpiSummaryFiltered", mock.Anything, mock.Anything, "").Return(database.KpiSummary{}, errTest)

	req, _ := http.NewRequest("GET", "/admin/analytics/summary", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusInternalServerError, s.recorder.Code)
}

func (s *ServerTestSuite) TestHandleRoutingDecisions() {
	decisions := []database.RoutingDecisionRow{
		{ID: 1, SelectedModel: "gpt-4-turbo", Status: "success"},
	}
	// Admin auth → userFilter=""
	s.mockDB.On("GetRoutingDecisionsFiltered", mock.Anything, mock.Anything, 5, 0, "").Return(decisions, 10, nil)

	req, _ := http.NewRequest("GET", "/admin/analytics/routing-decisions?range=7d&page=1&pageSize=5", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)

	var resp map[string]interface{}
	s.Require().NoError(json.Unmarshal(s.recorder.Body.Bytes(), &resp))
	s.Equal(float64(10), resp["total"])
	s.Equal(float64(1), resp["page"])
	s.Equal(float64(5), resp["pageSize"])
}

func (s *ServerTestSuite) TestHandleRoutingDecisions_DefaultPagination() {
	s.mockDB.On("GetRoutingDecisionsFiltered", mock.Anything, mock.Anything, 10, 0, "").Return([]database.RoutingDecisionRow{}, 0, nil)

	req, _ := http.NewRequest("GET", "/admin/analytics/routing-decisions", nil)
	req.Header.Set("X-API-Key", "test-admin-key")
	s.server.GetRouter().ServeHTTP(s.recorder, req)

	s.Equal(http.StatusOK, s.recorder.Code)

	var resp map[string]interface{}
	s.Require().NoError(json.Unmarshal(s.recorder.Body.Bytes(), &resp))
	s.Equal(float64(1), resp["page"])
	s.Equal(float64(10), resp["pageSize"])
}
