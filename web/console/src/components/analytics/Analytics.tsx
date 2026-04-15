import type { AnalyticsProps } from '@/types/analytics'
import { KpiCards } from './KpiCards'
import { TimeRangeSelector } from './TimeRangeSelector'
import { RoutingDecisionsList } from './RoutingDecisionsList'
import { ModelUsageCharts } from './ModelUsageCharts'
import { LatencySection } from './LatencySection'

export function Analytics({
  kpiSummary,
  routingDecisions,
  modelUsage,
  latencyPerModel,
  latencyTimeSeries,
  latencyPercentiles,
  onTimeRangeChange,
  onCustomDateRange,
  onViewDecision,
  onNextPage,
  onPrevPage,
}: AnalyticsProps) {
  return (
    <div className="p-5 lg:p-6 space-y-5">
      {/* Time range + KPIs */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <h2 className="text-lg font-semibold text-slate-800 dark:text-slate-200">
          Analytics
        </h2>
        <TimeRangeSelector
          onTimeRangeChange={onTimeRangeChange}
          onCustomDateRange={onCustomDateRange}
        />
      </div>

      <KpiCards summary={kpiSummary} />

      {/* Routing Decisions */}
      <RoutingDecisionsList
        decisions={routingDecisions}
        onViewDecision={onViewDecision}
        onNextPage={onNextPage}
        onPrevPage={onPrevPage}
      />

      {/* Charts row: Usage + Latency */}
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-5">
        <ModelUsageCharts usage={modelUsage} />
        <LatencySection
          perModel={latencyPerModel}
          timeSeries={latencyTimeSeries}
          percentiles={latencyPercentiles}
        />
      </div>
    </div>
  )
}
