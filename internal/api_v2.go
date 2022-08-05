package internal

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type CreateTestRunRequestData struct {
	ApiKey       string `json:"apiKey"`
	ScenarioName string `json:"scenarioName"`
	RunName      string `json:"runName"`
	Environment  string `json:"environment"`
}

type CreateTestRunRequest struct {
	Data *CreateTestRunRequestData `json:"data"`
}

type TestRun struct {
	ID           string `json:"id"`
	ScenarioName string `json:"scenarioName"`
	Environment  string `json:"environment"`
	WriteToken   string `json:"writeToken"`
}

type CreateTestRunResponse struct {
	Result struct {
		Success bool    `json:"success"`
		Data    TestRun `json:"data"`
	} `json:"result"`
}

type NewMetric struct {
	OperationName  string  `json:"operationName"`
	RequestCount   uint64  `json:"requestCount"`
	FailureCount   uint64  `json:"failureCount"`
	VirtualUserMax uint64  `json:"virtualUserMax"`
	LatencyAvgMs   float64 `json:"latencyAvgMs"`
	LatencyMinMs   float64 `json:"latencyMinMs"`
	LatencyMaxMs   float64 `json:"latencyMaxMs"`
	LatencyP50Ms   float64 `json:"latencyP50Ms"`
	LatencyP75Ms   float64 `json:"latencyP75Ms"`
	LatencyP90Ms   float64 `json:"latencyP90Ms"`
	LatencyP95Ms   float64 `json:"latencyP95Ms"`
	LatencyP99Ms   float64 `json:"latencyP99Ms"`
}

type NewChartMetric struct {
	Timestamp            uint64  `json:"timestamp"`
	TimeAggregationLevel string  `json:"timeAggregationLevel"`
	OperationName        string  `json:"operationName"`
	RequestCount         uint64  `json:"requestCount"`
	FailureCount         uint64  `json:"failureCount"`
	VirtualUserMax       uint64  `json:"virtualUserMax"`
	LatencyAvgMs         float64 `json:"latencyAvgMs"`
	LatencyMinMs         float64 `json:"latencyMinMs"`
	LatencyMaxMs         float64 `json:"latencyMaxMs"`
	LatencyP50Ms         float64 `json:"latencyP50Ms"`
	LatencyP75Ms         float64 `json:"latencyP75Ms"`
	LatencyP90Ms         float64 `json:"latencyP90Ms"`
	LatencyP95Ms         float64 `json:"latencyP95Ms"`
	LatencyP99Ms         float64 `json:"latencyP99Ms"`
}

type CreateTestChartMetricsRequestData struct {
	Token   string           `json:"token"`
	Metrics []NewChartMetric `json:"metrics"`
}

type CreateTestChartMetricsRequest struct {
	Data *CreateTestChartMetricsRequestData `json:"data"`
}

type CreateTestSummaryMetricsRequestData struct {
	Token   string      `json:"token"`
	Metrics []NewMetric `json:"metrics"`
}

type CreateTestSummaryMetricsRequest struct {
	Data *CreateTestSummaryMetricsRequestData `json:"data"`
}

func CreateTestRun(host string, apiKey string, name string) TestRun {
	postBody, err := json.Marshal(CreateTestRunRequest{
		Data: &CreateTestRunRequestData{
			ApiKey:       apiKey,
			ScenarioName: name,
		},
	})

	if err != nil {
		log.Fatalf("An error occurred while serializing request body %v", err)
	}

	resp, err := http.Post(host+"/v2/test.createRun", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll((resp.Body))
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalln("Request failed with:", string(body))
	}

	var parsed CreateTestRunResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		log.Println("Can not unmarshal JSON")
	}

	return parsed.Result.Data
}

func CreateTestChartMetrics(host string, token string, dataPoints []MetricDataPoint, dataPointsByLabel map[string][]MetricDataPoint) {
	batch := 200
	for i := 0; i < len(dataPoints); i += batch {
		j := i + batch
		if j > len(dataPoints) {
			j = len(dataPoints)
		}

		CreateTestChartMetricsBatch(host, token, dataPoints[i:j])
	}

	// TODO(bobsin): combine batches if less than 200
	for _, v := range dataPointsByLabel {
		for i := 0; i < len(v); i += batch {
			j := i + batch
			if j > len(v) {
				j = len(v)
			}

			CreateTestChartMetricsBatch(host, token, v[i:j])
		}
	}
}

func CreateTestChartMetricsBatch(host string, token string, dataPoints []MetricDataPoint) {
	postBody, err := json.Marshal(CreateTestChartMetricsRequest{
		Data: &CreateTestChartMetricsRequestData{
			Token:   token,
			Metrics: mapMetricDataPoints(dataPoints),
		},
	})

	if err != nil {
		log.Fatalf("An error occurred while serializing request body %v", err)
	}

	resp, err := http.Post(host+"/v2/test.createChartMetrics", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll((resp.Body))
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalln("Request failed with:", string(body))
	}
}

func CreateTestSummaryMetrics(host string, token string, metrics MetricSummary, metricsByLabel map[string]MetricSummary) {
	var mappedMetrics []NewMetric

	mappedMetrics = append(mappedMetrics, mapMetricSummary(metrics))
	for _, v := range metricsByLabel {
		mappedMetrics = append(mappedMetrics, mapMetricSummary(v))
	}

	postBody, err := json.Marshal(CreateTestSummaryMetricsRequest{
		Data: &CreateTestSummaryMetricsRequestData{
			Token:   token,
			Metrics: mappedMetrics,
		},
	})

	if err != nil {
		log.Fatalf("An error occurred while serializing request body %v", err)
	}

	resp, err := http.Post(host+"/v2/test.createSummaryMetrics", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll((resp.Body))
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalln("Request failed with:", string(body))
	}
}

func mapMetricDataPoints(dataPoints []MetricDataPoint) []NewChartMetric {
	result := make([]NewChartMetric, len(dataPoints))
	for i, dp := range dataPoints {
		result[i] = mapMetricDataPoint(dp)
	}
	return result
}

func mapMetricDataPoint(dp MetricDataPoint) NewChartMetric {
	return NewChartMetric{
		Timestamp:            dp.TimeStamp,
		TimeAggregationLevel: "5s",
		OperationName:        dp.Label,
		RequestCount:         dp.Requests,
		FailureCount:         dp.Failures,
		VirtualUserMax:       dp.VirtualUsers,
		LatencyAvgMs:         dp.Latencies.AvgMs,
		LatencyMinMs:         dp.Latencies.MinMs,
		LatencyMaxMs:         dp.Latencies.MaxMs,
		LatencyP50Ms:         dp.Latencies.P50Ms,
		LatencyP75Ms:         dp.Latencies.P75Ms,
		LatencyP90Ms:         dp.Latencies.P90Ms,
		LatencyP95Ms:         dp.Latencies.P95Ms,
		LatencyP99Ms:         dp.Latencies.P99Ms,
	}
}

func mapMetricSummary(summary MetricSummary) NewMetric {
	return NewMetric{
		OperationName:  summary.Label,
		RequestCount:   summary.TotalRequests,
		FailureCount:   summary.TotalFailures,
		VirtualUserMax: summary.MaxVirtualUsers,
		LatencyAvgMs:   summary.Latencies.AvgMs,
		LatencyMinMs:   summary.Latencies.MinMs,
		LatencyMaxMs:   summary.Latencies.MaxMs,
		LatencyP50Ms:   summary.Latencies.P50Ms,
		LatencyP75Ms:   summary.Latencies.P75Ms,
		LatencyP90Ms:   summary.Latencies.P90Ms,
		LatencyP95Ms:   summary.Latencies.P95Ms,
		LatencyP99Ms:   summary.Latencies.P99Ms,
	}
}
