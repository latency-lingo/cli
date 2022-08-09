package internal

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type CreateReportResponse struct {
	Result struct {
		Success bool `json:"success"`
		Data    struct {
			ID         string `json:"id"`
			WriteToken string `json:"writeToken"`
			Label      string `json:"label"`
			Metrics    struct {
				Latencies struct {
					AvgMs int `json:"avgMs"`
				} `json:"latencies"`
				TotalRequests   int `json:"totalRequests"`
				TotalFailures   int `json:"totalFailures"`
				MaxVirtualUsers int `json:"maxVirtualUsers"`
			} `json:"metrics"`
			CreatedAt int `json:"createdAt"`
		} `json:"data"`
	} `json:"result"`
}

type PublishDataPointsRequestData struct {
	ReportUUID        string                       `json:"reportUuid"`
	Token             string                       `json:"token"`
	Action            string                       `json:"action"`
	DataPoints        []MetricDataPoint            `json:"dataPoints"`
	DataPointsByLabel map[string][]MetricDataPoint `json:"dataPointsByLabel"`
}

type PublishDataPointsRequest struct {
	Data *PublishDataPointsRequestData `json:"data"`
}

type PublishMetricSummaryRequestData struct {
	ReportUUID     string                   `json:"reportUuid"`
	Token          string                   `json:"token"`
	Metrics        MetricSummary            `json:"metrics"`
	MetricsByLabel map[string]MetricSummary `json:"metricsByLabel"`
}

type PublishMetricSummaryRequest struct {
	Data PublishMetricSummaryRequestData `json:"data"`
}

func CreateReport(host string, apiKey string, label string) (*CreateReportResponse, error) {
	postBody, err := json.Marshal(map[string]map[string]string{
		"data": {
			"apiKey": apiKey,
			"label":  label,
		},
	})

	if err != nil {
		return nil, errors.Errorf("[reports.create] failed to build request body: %w", err)
	}

	resp, err := http.Post(host+"/v1/reports.create", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		return nil, errors.Errorf("[reports.create] request failed: %w", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("[reports.create] failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("[reports.create] request failed: %s", string(body))
	}

	var parsed CreateReportResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errors.Errorf("[reports.create] failed to parse response: %w", err)
	}

	return &parsed, nil
}

func PublishDataPoints(host string, reportId string, reportToken string, dataPoints []MetricDataPoint, dataPointsByLabel map[string][]MetricDataPoint) (bool, error) {
	batch := 200
	for i := 0; i < len(dataPoints); i += batch {
		j := i + batch
		if j > len(dataPoints) {
			j = len(dataPoints)
		}

		if _, err := PublishDataPointsBatch(host, reportId, reportToken, dataPoints[i:j], make(map[string][]MetricDataPoint)); err != nil {
			return false, err
		}
	}

	// TODO(bobsin): combine batches if less than 200
	for k, v := range dataPointsByLabel {
		for i := 0; i < len(v); i += batch {
			j := i + batch
			if j > len(v) {
				j = len(v)
			}

			dpByLabelBatch := make(map[string][]MetricDataPoint)
			dpByLabelBatch[k] = v[i:j]

			if _, err := PublishDataPointsBatch(host, reportId, reportToken, []MetricDataPoint{}, dpByLabelBatch); err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

func PublishDataPointsBatch(host string, reportId string, reportToken string, dataPoints []MetricDataPoint, dataPointsByLabel map[string][]MetricDataPoint) (bool, error) {
	postBody, err := json.Marshal(PublishDataPointsRequest{
		Data: &PublishDataPointsRequestData{
			ReportUUID:        reportId,
			Token:             reportToken,
			Action:            "add",
			DataPoints:        dataPoints,
			DataPointsByLabel: dataPointsByLabel,
		},
	})

	if err != nil {
		return false, errors.Errorf("[reports.updateData] failed to build request body: %w", err)
	}

	resp, err := http.Post(host+"/v1/reports.updateData", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		return false, errors.Errorf("[reports.updateData] request failed: %w", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, errors.Errorf("[reports.updateData] failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, errors.Errorf("[reports.updateData] request failed: %s", string(body))
	}

	return true, nil
}

func PublishMetricSummary(host string, reportId string, reportToken string, metrics MetricSummary, metricsByLabel map[string]MetricSummary) (bool, error) {
	postBody, err := json.Marshal(PublishMetricSummaryRequest{
		Data: PublishMetricSummaryRequestData{
			ReportUUID:     reportId,
			Token:          reportToken,
			Metrics:        metrics,
			MetricsByLabel: metricsByLabel,
		},
	})

	if err != nil {
		return false, errors.Errorf("[reports.update] failed to build request body: %w", err)
	}

	resp, err := http.Post(host+"/v1/reports.update", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		return false, errors.Errorf("[reports.update] request failed: %w", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, errors.Errorf("[reports.update] failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, errors.Errorf("[reports.update] request failed: %s", string(body))
	}

	return true, nil
}
