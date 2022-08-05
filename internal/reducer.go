package internal

import "github.com/montanaflynn/stats"

type GlobalDataCounter struct {
	Label           string
	TotalRequests   uint64
	TotalFailures   uint64
	MaxVirtualUsers uint64
	RawLatencies    []float64
}

type LabeledDataCounter = map[string]*GlobalDataCounter

type GroupedResult struct {
	DataPoints        []MetricDataPoint
	DataPointsByLabel map[string][]MetricDataPoint
}

var globalDataCounter = &GlobalDataCounter{}
var labeledDataCounter = make(map[string]*GlobalDataCounter)
var globalCountersFull = false
var allTimeAggregationLevels = []TimeAggregationLevel{
	FiveSeconds,
	ThirtySeconds,
	OneMinute,
	FiveMinutes,
	ThirtyMinutes,
}

func GroupAllDataPoints(ungrouped []UngroupedMetricDataPoint) GroupedResult {
	groupedResult := GroupedResult{}
	groupedResult.DataPointsByLabel = make(map[string][]MetricDataPoint)

	// TODO(bobsin): disqualify levels based on duration eg. duration / timeAggregationLevel < 1000
	for _, timeAggregationLevel := range allTimeAggregationLevels {
		localResult := GroupDataPoints(ungrouped, timeAggregationLevel)
		groupedResult.DataPoints = append(groupedResult.DataPoints, localResult.DataPoints...)
		for label, dataPoints := range localResult.DataPointsByLabel {
			groupedResult.DataPointsByLabel[label] = append(groupedResult.DataPointsByLabel[label], dataPoints...)
		}
		globalCountersFull = true
	}

	return groupedResult
}

func GroupDataPoints(ungrouped []UngroupedMetricDataPoint, timeAggregationLevel TimeAggregationLevel) GroupedResult {
	var (
		startTime         uint64
		dataPoints        []MetricDataPoint
		dataPointsByLabel map[string][]MetricDataPoint
		batch             []UngroupedMetricDataPoint
		batchByLabel      map[string][]UngroupedMetricDataPoint
	)
	dataPointsByLabel = make(map[string][]MetricDataPoint)
	batchByLabel = make(map[string][]UngroupedMetricDataPoint)

	for _, dp := range ungrouped {
		if startTime == 0 {
			// init start time to last 5s interval from time stamp
			startTime = calculateIntervalFloor(dp.TimeStamp, timeAggregationLevel.Seconds())
		}

		if dp.TimeStamp-startTime > timeAggregationLevel.Seconds() {
			dataPoints = append(dataPoints, groupDataPointBatch(batch, startTime, "", timeAggregationLevel))
			mergeDataPointsByLabel(dataPointsByLabel, batchByLabel, startTime, timeAggregationLevel)
			batch = nil
			batchByLabel = map[string][]UngroupedMetricDataPoint{}
			startTime = 0
		}

		batch = append(batch, dp)
		batchByLabel[dp.Label] = append(batchByLabel[dp.Label], dp)
	}

	if len(batch) > 0 {
		startTime = calculateIntervalFloor(batch[0].TimeStamp, timeAggregationLevel.Seconds())
		dataPoints = append(dataPoints, groupDataPointBatch(batch, startTime, "", timeAggregationLevel))
		mergeDataPointsByLabel(dataPointsByLabel, batchByLabel, startTime, timeAggregationLevel)
	}

	return GroupedResult{
		DataPoints:        dataPoints,
		DataPointsByLabel: dataPointsByLabel,
	}
}

func mergeDataPointsByLabel(existing map[string][]MetricDataPoint, batch map[string][]UngroupedMetricDataPoint, startTime uint64, timeAggregationLevel TimeAggregationLevel) {
	for label, dataPoints := range batch {
		existing[label] = append(existing[label], groupDataPointBatch(dataPoints, startTime, label, timeAggregationLevel))
	}
}

func groupDataPointBatch(ungrouped []UngroupedMetricDataPoint, startTime uint64, label string, timeAggregationLevel TimeAggregationLevel) MetricDataPoint {
	var (
		latencies []float64
		grouped   MetricDataPoint
	)

	grouped.TimeStamp = startTime
	grouped.TimeAggregationLevel = timeAggregationLevel

	for _, dp := range ungrouped {
		if label != "" {
			grouped.Label = label
		}

		grouped.Requests += dp.Requests
		grouped.Failures += dp.Failures

		if dp.VirtualUsers > grouped.VirtualUsers {
			grouped.VirtualUsers = dp.VirtualUsers
		}

		latencies = append(latencies, float64(dp.Latency))
	}

	grouped.Latencies = calculateLatencySummary(latencies)

	if label != "" {
		updateGlobalCounter(globalDataCounter, grouped, latencies)
		if labeledDataCounter[label] == nil {
			labeledDataCounter[label] = &GlobalDataCounter{}
		}

		updateGlobalCounter(labeledDataCounter[label], grouped, latencies)
	}

	return grouped
}

func updateGlobalCounter(counter *GlobalDataCounter, metric MetricDataPoint, latencies []float64) {
	if globalCountersFull {
		return
	}

	counter.TotalRequests += metric.Requests
	counter.TotalFailures += metric.Failures
	if metric.VirtualUsers > counter.MaxVirtualUsers {
		counter.MaxVirtualUsers = metric.VirtualUsers
	}
	counter.RawLatencies = append(counter.RawLatencies, latencies...)
}

func calculateLatencySummary(latencies []float64) *Latencies {
	summary := Latencies{}
	summary.AvgMs, _ = stats.Mean(latencies)
	summary.MaxMs, _ = stats.Max(latencies)
	summary.MinMs, _ = stats.Min(latencies)
	summary.P50Ms, _ = stats.Percentile(latencies, 50)
	summary.P75Ms, _ = stats.Percentile(latencies, 75)
	summary.P90Ms, _ = stats.Percentile(latencies, 90)
	summary.P95Ms, _ = stats.Percentile(latencies, 95)
	summary.P99Ms, _ = stats.Percentile(latencies, 99)

	summary.AvgMs, _ = stats.Round(summary.AvgMs, 2)
	summary.MaxMs, _ = stats.Round(summary.MaxMs, 2)
	summary.MinMs, _ = stats.Round(summary.MinMs, 2)
	summary.P50Ms, _ = stats.Round(summary.P50Ms, 2)
	summary.P75Ms, _ = stats.Round(summary.P75Ms, 2)
	summary.P90Ms, _ = stats.Round(summary.P90Ms, 2)
	summary.P95Ms, _ = stats.Round(summary.P95Ms, 2)
	summary.P99Ms, _ = stats.Round(summary.P99Ms, 2)

	return &summary
}

func CalculateMetricSummaryOverall() MetricSummary {
	return calculateMetricSummary(globalDataCounter, "")
}

func CalculateMetricSummaryByLabel() map[string]MetricSummary {
	metricSummaryByLabel := make(map[string]MetricSummary)
	for label, counter := range labeledDataCounter {
		metricSummaryByLabel[label] = calculateMetricSummary(counter, label)
	}

	return metricSummaryByLabel
}

func calculateMetricSummary(globalDataCounter *GlobalDataCounter, label string) MetricSummary {
	summary := MetricSummary{}

	if label != "" {
		summary.Label = label
	}

	summary.TotalRequests = globalDataCounter.TotalRequests
	summary.TotalFailures = globalDataCounter.TotalFailures
	summary.MaxVirtualUsers = globalDataCounter.MaxVirtualUsers
	summary.Latencies = calculateLatencySummary(globalDataCounter.RawLatencies)

	return summary
}

func calculateIntervalFloor(timeStamp uint64, timeAggSeconds uint64) uint64 {
	difference := timeStamp % 60 % timeAggSeconds
	return timeStamp - difference
}
