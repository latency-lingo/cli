package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

type CreateTestRunRequestData struct {
	ApiKey          string `json:"apiKey"`
	ScenarioName    string `json:"scenarioName"`
	RunName         string `json:"runName"`
	Environment     string `json:"environment"`
	StartedAt       uint64 `json:"startedAt"`
	StoppedAt       uint64 `json:"stoppedAt"`
	PublishStrategy string `json:"publishStrategy"`
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

type LingoSample struct {
	TimeStamp       uint64 `json:"timestamp"`
	Label           string `json:"label"`
	Elapsed         uint64 `json:"elapsed"`
	ResponseCode    int    `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
	ThreadName      string `json:"threadName"`
	DataType        string `json:"dataType"`
	Success         bool   `json:"success"`
	FailureMessage  string `json:"failureMessage"`
	Bytes           int    `json:"bytes"`
	SentBytes       int    `json:"sentBytes"`
	GrpThreads      int    `json:"grpThreads"`
	AllThreads      int    `json:"allThreads"`
	URL             string `json:"url"`
	IdleTime        uint64 `json:"idleTime"`
	Connect         uint64 `json:"connect"`
}

type CreateTestSamplesRequestData struct {
	Token   string        `json:"token"`
	Samples []LingoSample `json:"samples"`
}

type CreateTestSamplesRequest struct {
	Data *CreateTestSamplesRequestData `json:"data"`
}

type UpdateTestRunRequestData struct {
	Token     string `json:"token"`
	StoppedAt uint64 `json:"stoppedAt"`
}

type UpdateTestRunRequest struct {
	Data *UpdateTestRunRequestData `json:"data"`
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

type TimeAggregationLevel string

const (
	Undefined     TimeAggregationLevel = ""
	FiveSeconds   TimeAggregationLevel = "5s"
	ThirtySeconds TimeAggregationLevel = "30s"
	OneMinute     TimeAggregationLevel = "1m"
	FiveMinutes   TimeAggregationLevel = "5m"
	ThirtyMinutes TimeAggregationLevel = "30m"
)

func (t TimeAggregationLevel) Seconds() uint64 {
	switch t {
	case FiveSeconds:
		return 5
	case ThirtySeconds:
		return 30
	case OneMinute:
		return 60
	case FiveMinutes:
		return 300
	case ThirtyMinutes:
		return 1800
	default:
		return 5
	}
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

type GetTestRunResultsRequestData struct {
	Token string `json:"token"`
}

type GetTestRunResultsRequest struct {
	Data *GetTestRunResultsRequestData `json:"data"`
}

type RunResultData struct {
	RunId      string `json:"runId"`
	Status     string `json:"status"`
	Thresholds []struct {
		Status      string `json:"status"`
		Description string `json:"description"`
	} `json:"thresholds"`
}

type GetTestRunResultsResponse struct {
	Result struct {
		Success bool           `json:"success"`
		Data    *RunResultData `json:"data"`
	} `json:"result"`
}

func CreateTestRun(host string, apiKey string, name string, startedAt uint64, stoppedAt uint64, publishStrategy string) (*TestRun, error) {
	span := sentry.StartSpan(context.Background(), "CreateTestRun")
	defer span.Finish()

	postBody, err := json.Marshal(CreateTestRunRequest{
		Data: &CreateTestRunRequestData{
			ApiKey:          apiKey,
			ScenarioName:    name,
			StartedAt:       startedAt,
			StoppedAt:       stoppedAt,
			PublishStrategy: publishStrategy,
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "[test.createRun] failed to build request body")
	}

	resp, err := http.Post(host+"/v2/test.createRun", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		return nil, errors.Wrap(err, "[test.createRun] request failed")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll((resp.Body))
	if err != nil {
		return nil, errors.Wrap(err, "[test.createRun] failed to parse response")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("[test.createRun] request failed: %s", string(body))
	}

	var parsed CreateTestRunResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errors.Wrap(err, "[test.createRun] failed to parse response")
	}

	return &parsed.Result.Data, nil
}

func CreateTestChartMetrics(host string, token string, dataPoints []MetricDataPoint, dataPointsByLabel map[string][]MetricDataPoint) (bool, error) {
	span := sentry.StartSpan(context.Background(), "CreateTestChartMetrics")
	defer span.Finish()

	allDataPoints := make([]MetricDataPoint, 0)
	allDataPoints = append(allDataPoints, dataPoints...)
	for _, v := range dataPointsByLabel {
		allDataPoints = append(allDataPoints, v...)
	}

	batch := 500
	for i := 0; i < len(allDataPoints); i += batch {
		j := i + batch
		if j > len(allDataPoints) {
			j = len(allDataPoints)
		}

		if _, err := CreateTestChartMetricsBatch(host, token, allDataPoints[i:j]); err != nil {
			return false, err
		}
	}

	return true, nil
}

func CreateTestChartMetricsBatch(host string, token string, dataPoints []MetricDataPoint) (bool, error) {
	postBody, err := json.Marshal(CreateTestChartMetricsRequest{
		Data: &CreateTestChartMetricsRequestData{
			Token:   token,
			Metrics: mapMetricDataPoints(dataPoints),
		},
	})

	if err != nil {
		return false, errors.Wrap(err, "[test.createChartMetrics] failed to build request body")
	}

	resp, err := http.Post(host+"/v2/test.createChartMetrics", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		return false, errors.Wrap(err, "[test.createChartMetrics] request failed")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll((resp.Body))
	if err != nil {
		return false, errors.Wrap(err, "[test.createChartMetrics] failed to parse response")
	}

	if resp.StatusCode != http.StatusOK {
		return false, errors.Errorf("[test.createChartMetrics] request failed: %s", string(body))
	}

	return true, nil
}

func CreateTestSummaryMetrics(host string, token string, metrics MetricSummary, metricsByLabel map[string]MetricSummary) (bool, error) {
	span := sentry.StartSpan(context.Background(), "CreateTestSummaryMetrics")
	defer span.Finish()

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
		return false, errors.Wrap(err, "[test.createSummaryMetrics] failed to build request body")
	}

	resp, err := http.Post(host+"/v2/test.createSummaryMetrics", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		return false, errors.Wrap(err, "[test.createSummaryMetrics] request failed")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll((resp.Body))
	if err != nil {
		return false, errors.Wrap(err, "[test.createSummaryMetrics] request failed")
	}

	if resp.StatusCode != http.StatusOK {
		return false, errors.Errorf("[test.createSummaryMetrics] request failed: %s", string(body))
	}

	return true, nil
}

func CreateTestSamples(host string, token string, samples []LingoSample) (bool, error) {
	span := sentry.StartSpan(context.Background(), "CreateTestSamples")
	defer span.Finish()

	// TODO(bobsin): parallelize this
	batch := 500
	for i := 0; i < len(samples); i += batch {
		j := i + batch
		if j > len(samples) {
			j = len(samples)
		}

		if _, err := CreateTestSamplesBatch(host, token, samples[i:j]); err != nil {
			return false, err
		}
	}

	return true, nil
}

func CreateTestSamplesBatch(host string, token string, samples []LingoSample) (bool, error) {
	postBody, err := json.Marshal(CreateTestSamplesRequest{
		Data: &CreateTestSamplesRequestData{
			Token:   token,
			Samples: samples,
		},
	})

	if err != nil {
		return false, errors.Wrap(err, "[test.createSamples] failed to build request body")
	}

	resp, err := http.Post(host+"/v2/test.createSamples", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		return false, errors.Wrap(err, "[test.createSamples] request failed")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll((resp.Body))
	if err != nil {
		return false, errors.Wrap(err, "[test.createSamples] failed to parse response")
	}

	if resp.StatusCode != http.StatusOK {
		return false, errors.Errorf("[test.createSamples] request failed: %s", string(body))
	}

	return true, nil
}

func UpdateTestRun(host string, token string, stoppedAt uint64) (bool, error) {
	postBody, err := json.Marshal(UpdateTestRunRequest{
		Data: &UpdateTestRunRequestData{
			Token:     token,
			StoppedAt: stoppedAt,
		},
	})

	if err != nil {
		return false, errors.Wrap(err, "[test.updateRun] failed to build request body")
	}

	resp, err := http.Post(host+"/v2/test.updateRun", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		return false, errors.Wrap(err, "[test.updateRun] request failed")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll((resp.Body))
	if err != nil {
		return false, errors.Wrap(err, "[test.updateRun] failed to parse response")
	}

	if resp.StatusCode != http.StatusOK {
		return false, errors.Errorf("[test.updateRun] request failed: %s", string(body))
	}

	return true, nil
}

func GetTestRunResults(host string, token string) (*RunResultData, error) {
	postBody, err := json.Marshal(GetTestRunResultsRequest{
		Data: &GetTestRunResultsRequestData{
			Token: token,
		},
	})

	if err != nil {
		return nil, errors.Wrap(err, "[test.getRunResults] failed to build request body")
	}

	resp, err := http.Post(host+"/v2/test.getRunResults", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		return nil, errors.Wrap(err, "[test.getRunResults] request failed")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll((resp.Body))
	if err != nil {
		return nil, errors.Wrap(err, "[test.getRunResults] failed to parse response")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("[test.getRunResults] request failed: %s", string(body))
	}

	var parsed GetTestRunResultsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errors.Wrap(err, "[test.getRunResults] failed to parse response")
	}

	return parsed.Result.Data, nil
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
		TimeAggregationLevel: string(dp.TimeAggregationLevel),
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
