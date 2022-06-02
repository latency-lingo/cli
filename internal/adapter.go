package internal

import (
	"log"
	"math"
	"strconv"
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
	Label        string     `json:"label"`
	Requests     uint64     `json:"requests"`
	Failures     uint64     `json:"failures"`
	VirtualUsers uint64     `json:"virtualUsers"`
	TimeStamp    uint64     `json:"timeStamp"`
	Latencies    *Latencies `json:"latencies"`
}

type UngroupedMetricDataPoint struct {
	Requests     uint64
	Failures     uint64
	VirtualUsers uint64
	TimeStamp    uint64
	Latency      uint64
	Label        string
}

func TranslateJmeterRow(row []string) UngroupedMetricDataPoint {
	var (
		failures uint64
	)

	if row[7] == "true" {
		failures = 0
	} else {
		failures = 1
	}

	// TODO(bobsin): improve error handling
	virtualUsers, err := strconv.ParseUint(row[12], 10, 32)
	if err != nil {
		log.Println(err)
	}

	timeStamp, err := strconv.ParseUint(row[0], 10, 64)
	if err != nil {
		log.Println(err)
	}

	latency, err := strconv.ParseUint(row[1], 10, 32)
	if err != nil {
		log.Println(err)
	}

	parsed := UngroupedMetricDataPoint{
		Label:        row[2],
		Requests:     1,
		Failures:     failures,
		VirtualUsers: virtualUsers,
		TimeStamp:    uint64(math.Round(float64(timeStamp / 1000.0))),
		Latency:      latency,
	}

	return parsed
}
