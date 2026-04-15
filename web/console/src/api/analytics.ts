import { apiGet } from './client'
import type {
  KpiSummary,
  RoutingDecision,
  ModelUsageEntry,
  LatencyPerModelEntry,
  LatencyTimeSeriesPoint,
  LatencyPercentilesEntry,
  TimeRange,
} from '@/types/analytics'

interface AnalyticsSummaryResponse {
  kpiSummary: KpiSummary
  modelUsage: ModelUsageEntry[]
  latencyPerModel: LatencyPerModelEntry[]
  latencyTimeSeries: LatencyTimeSeriesPoint[]
  latencyPercentiles: LatencyPercentilesEntry[]
}

interface RoutingDecisionsResponse {
  decisions: RoutingDecision[]
  total: number
  page: number
  pageSize: number
}

function buildTimeParams(
  range_: TimeRange,
  start?: string,
  end?: string
): string {
  const params = new URLSearchParams()
  if (range_ === 'custom' && start && end) {
    params.set('start', start)
    params.set('end', end)
  } else {
    params.set('range', range_)
  }
  return params.toString()
}

export async function fetchAnalyticsSummary(
  range_: TimeRange,
  start?: string,
  end?: string
): Promise<AnalyticsSummaryResponse> {
  const qs = buildTimeParams(range_, start, end)
  return apiGet<AnalyticsSummaryResponse>(
    `/admin/analytics/summary?${qs}`
  )
}

export async function fetchRoutingDecisions(
  range_: TimeRange,
  page: number,
  pageSize: number,
  start?: string,
  end?: string
): Promise<RoutingDecisionsResponse> {
  const params = new URLSearchParams()
  if (range_ === 'custom' && start && end) {
    params.set('start', start)
    params.set('end', end)
  } else {
    params.set('range', range_)
  }
  params.set('page', String(page))
  params.set('pageSize', String(pageSize))
  return apiGet<RoutingDecisionsResponse>(
    `/admin/analytics/routing-decisions?${params.toString()}`
  )
}
