package internal

import (
	"errors"
	"log"
	"strconv"
	"strings"
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

type ColumnIndices struct {
	Success         uint32
	SuccessFound    bool
	AllThreads      uint32
	AllThreadsFound bool
	Elapsed         uint32
	ElapsedFound    bool
	TimeStamp       uint32
	TimeStampFound  bool
	Label           uint32
	LabelFound      bool
}

func BuildColumnIndices(row []string) (*ColumnIndices, error) {
	var indices ColumnIndices
	for i, column := range row {
		switch column {
		case "success":
			indices.Success = uint32(i)
			indices.SuccessFound = true
		case "allThreads":
			indices.AllThreads = uint32(i)
			indices.AllThreadsFound = true
		case "elapsed":
			indices.Elapsed = uint32(i)
			indices.ElapsedFound = true
		case "timeStamp":
			indices.TimeStamp = uint32(i)
			indices.TimeStampFound = true
		case "label":
			indices.Label = uint32(i)
			indices.LabelFound = true
		}
	}

	missing := []string{}
	if !indices.SuccessFound {
		missing = append(missing, "success")
	}
	if !indices.AllThreadsFound {
		missing = append(missing, "allThreads")
	}
	if !indices.ElapsedFound {
		missing = append(missing, "elapsed")
	}
	if !indices.TimeStampFound {
		missing = append(missing, "timeStamp")
	}
	if !indices.LabelFound {
		missing = append(missing, "label")
	}

	if len(missing) > 0 {
		return nil, errors.New("missing column(s): " + strings.Join(missing, ", "))
	}

	return &indices, nil
}

func TranslateJmeterRow(row []string, indices *ColumnIndices) UngroupedMetricDataPoint {
	var (
		failures uint64
	)

	if row[indices.Success] == "true" {
		failures = 0
	} else {
		failures = 1
	}

	// TODO(bobsin): improve error handling
	virtualUsers, err := strconv.ParseUint(row[indices.AllThreads], 10, 32)
	if err != nil {
		log.Println(err)
	}

	latency, err := strconv.ParseUint(row[indices.Elapsed], 10, 32)
	if err != nil {
		log.Println(err)
	}

	parsed := UngroupedMetricDataPoint{
		Label:        row[indices.Label],
		Requests:     1,
		Failures:     failures,
		VirtualUsers: virtualUsers,
		TimeStamp:    ParseTimeStamp(row[indices.TimeStamp]),
		Latency:      latency,
	}

	return parsed
}

var possibleTsFormats = []string{
	"2006-01-02 15:04:05.999",
	"2006/01/02 15:04:05.999",
	time.RFC3339,
	time.RFC3339Nano,
}

func ParseTimeStamp(timeStamp string) uint64 {
	parsed, err := strconv.ParseUint(timeStamp, 10, 64)
	if err == nil {
		// handle the case where the timestamp is in milliseconds
		if parsed > uint64(time.Now().Unix()+1000) {
			return uint64(time.UnixMilli(int64(parsed)).Unix())
		}

		return parsed
	}

	for _, tsFormat := range possibleTsFormats {
		parsed, err := time.ParseInLocation(tsFormat, timeStamp, time.Local)
		if err == nil {
			return uint64(parsed.Unix())
		}
	}

	sentry.CaptureMessage("unable to parse timestamp: " + timeStamp)
	log.Fatalf("unable to parse timestamp: %s", timeStamp)
	return 0
}
