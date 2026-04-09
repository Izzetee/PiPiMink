import { useState, useEffect, useCallback } from 'react'
import { Analytics } from '@/components/analytics'
import { fetchAnalyticsSummary, fetchRoutingDecisions } from '@/api'
import type {
  TimeRange,
  CustomDateRange,
  KpiSummary,
  RoutingDecision,
  ModelUsageEntry,
  LatencyPerModelEntry,
  LatencyTimeSeriesPoint,
  LatencyPercentilesEntry,
} from '@/types/analytics'

const PAGE_SIZE = 10

const emptyKpi: KpiSummary = {
  totalRequests: 0,
  avgLatencyMs: 0,
  mostUsedModel: '—',
  errorRate: 0,
}

export function AnalyticsPage() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [timeRange, setTimeRange] = useState<TimeRange>('24h')
  const [customRange, setCustomRange] = useState<CustomDateRange | null>(null)

  const [kpiSummary, setKpiSummary] = useState<KpiSummary>(emptyKpi)
  const [routingDecisions, setRoutingDecisions] = useState<RoutingDecision[]>([])
  const [modelUsage, setModelUsage] = useState<ModelUsageEntry[]>([])
  const [latencyPerModel, setLatencyPerModel] = useState<LatencyPerModelEntry[]>([])
  const [latencyTimeSeries, setLatencyTimeSeries] = useState<LatencyTimeSeriesPoint[]>([])
  const [latencyPercentiles, setLatencyPercentiles] = useState<LatencyPercentilesEntry[]>([])

  const [page, setPage] = useState(1)

  const fetchAll = useCallback(
    async (range: TimeRange, custom: CustomDateRange | null, pg: number) => {
      setLoading(true)
      setError(null)
      try {
        const start = custom?.start
        const end = custom?.end
        const [summary, decisions] = await Promise.all([
          fetchAnalyticsSummary(range, start, end),
          fetchRoutingDecisions(range, pg, PAGE_SIZE, start, end),
        ])
        setKpiSummary(summary.kpiSummary)
        setModelUsage(summary.modelUsage)
        setLatencyPerModel(summary.latencyPerModel)
        setLatencyTimeSeries(summary.latencyTimeSeries)
        setLatencyPercentiles(summary.latencyPercentiles)
        setRoutingDecisions(decisions.decisions)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load analytics')
      } finally {
        setLoading(false)
      }
    },
    []
  )

  useEffect(() => {
    fetchAll(timeRange, customRange, page)
  }, [fetchAll, timeRange, customRange, page])

  function handleTimeRangeChange(range: TimeRange) {
    setTimeRange(range)
    setCustomRange(null)
    setPage(1)
  }

  function handleCustomDateRange(range: CustomDateRange) {
    setTimeRange('custom')
    setCustomRange(range)
    setPage(1)
  }

  function handleNextPage() {
    setPage((p) => p + 1)
  }

  function handlePrevPage() {
    setPage((p) => Math.max(1, p - 1))
  }

  if (loading) {
    return (
      <div className="p-6 flex items-center justify-center min-h-[300px]">
        <div className="flex items-center gap-3 text-slate-500 dark:text-slate-400">
          <div className="w-5 h-5 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
          <span className="text-sm">Loading analytics...</span>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="rounded-xl border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/30 p-4 text-sm text-red-700 dark:text-red-300">
          {error}
        </div>
      </div>
    )
  }

  return (
    <Analytics
      kpiSummary={kpiSummary}
      routingDecisions={routingDecisions}
      modelUsage={modelUsage}
      latencyPerModel={latencyPerModel}
      latencyTimeSeries={latencyTimeSeries}
      latencyPercentiles={latencyPercentiles}
      onTimeRangeChange={handleTimeRangeChange}
      onCustomDateRange={handleCustomDateRange}
      onNextPage={handleNextPage}
      onPrevPage={handlePrevPage}
    />
  )
}
