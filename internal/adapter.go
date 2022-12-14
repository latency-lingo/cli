package internal

import (
	"log"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
)

type MetricSummary struct {
	Label           string     `json:"label"`
	Latencies       *Latencies `json:"latencies"`
	TotalRequests   uint64     `json:"totalRequests"`
	TotalFailures   uint64     `json:"totalFailures"`
	MaxVirtualUsers uint64     `json:"maxVirtualUsers"`
}

type Latencies struct {
	AvgMs float64 `json:"avgMs"`
	MinMs float64 `json:"minMs"`
	MaxMs float64 `json:"maxMs"`
	P50Ms float64 `json:"p50Ms"`
	P75Ms float64 `json:"p75Ms"`
	P90Ms float64 `json:"p90Ms"`
	P95Ms float64 `json:"p95Ms"`
	P99Ms float64 `json:"p99Ms"`
}

type MetricDataPoint struct {
	Label                string `json:"label"`
	Requests             uint64 `json:"requests"`
	Failures             uint64 `json:"failures"`
	VirtualUsers         uint64 `json:"virtualUsers"`
	TimeStamp            uint64 `json:"timeStamp"`
	TimeAggregationLevel TimeAggregationLevel
	Latencies            *Latencies `json:"latencies"`
}

type UngroupedMetricDataPoint struct {
	Requests     uint64
	Failures     uint64
	VirtualUsers uint64
	TimeStamp    uint64
	Latency      uint64
	Label        string
}

var possibleTsFormats = []string{
	"2006-01-02 15:04:05.999",
	"2006/01/02 15:04:05.999",
	time.RFC3339,
	time.RFC3339Nano,
	// k6 formatting
	"2006-01-02T15:04:05.999999-07:00",
}

func ParseTimeStampMillis(timeStamp string) uint64 {
	parsed, err := strconv.ParseUint(timeStamp, 10, 64)

	if err == nil {
		// handle the case where the timestamp is in seconds
		if parsed < 10000000000 {
			parsed = parsed * 1000
		}
		return parsed
	}

	for _, tsFormat := range possibleTsFormats {
		parsed, err := time.ParseInLocation(tsFormat, timeStamp, time.Local)
		if err == nil {
			return uint64(parsed.UnixMilli())
		}
	}

	sentry.CaptureMessage("unable to parse timestamp: " + timeStamp)
	log.Fatalf("unable to parse timestamp: %s", timeStamp)
	return 0
}
