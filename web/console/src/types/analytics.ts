// =============================================================================
// UI Data Shapes — These define the data the Analytics components expect
// =============================================================================

export interface KpiSummary {
  totalRequests: number
  avgLatencyMs: number
  mostUsedModel: string
  errorRate: number
}

export interface RoutingDecision {
  id: string
  timestamp: string
  promptSnippet: string
  fullPrompt: string
  analyzedTags: string[]
  tagRelevance: Record<string, number>
  selectedModel: string
  provider: string
  routingReason: string
  evaluatorModel: string
  evaluationTimeMs: number
  cacheHit: boolean
  latencyMs: number
  status: 'success' | 'error'
}

export interface ModelUsageEntry {
  modelName: string
  requestCount: number
  percentage: number
}

export interface LatencyPerModelEntry {
  modelName: string
  avgLatencyMs: number
}

export interface LatencyTimeSeriesPoint {
  timestamp: string
  avgLatencyMs: number
  p95LatencyMs: number
}

export interface LatencyPercentilesEntry {
  modelName: string
  p50Ms: number
  p95Ms: number
  p99Ms: number
}

export type TimeRange = '1h' | '24h' | '7d' | '30d' | 'custom'

export interface CustomDateRange {
  start: string
  end: string
}

// =============================================================================
// Component Props
// =============================================================================

export interface AnalyticsProps {
  kpiSummary: KpiSummary
  routingDecisions: RoutingDecision[]
  modelUsage: ModelUsageEntry[]
  latencyPerModel: LatencyPerModelEntry[]
  latencyTimeSeries: LatencyTimeSeriesPoint[]
  latencyPercentiles: LatencyPercentilesEntry[]
  /** Called when the user changes the time range preset */
  onTimeRangeChange?: (range: TimeRange) => void
  /** Called when the user selects a custom date range */
  onCustomDateRange?: (range: CustomDateRange) => void
  /** Called when the user clicks a routing decision to view details */
  onViewDecision?: (id: string) => void
  /** Called when the user navigates to the next page of routing decisions */
  onNextPage?: () => void
  /** Called when the user navigates to the previous page of routing decisions */
  onPrevPage?: () => void
}
