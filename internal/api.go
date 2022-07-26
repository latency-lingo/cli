package internal

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
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

func CreateReport(host string, apiKey string, label string) CreateReportResponse {
	postBody, err := json.Marshal(map[string]map[string]string{
		"data": {
			"apiKey": apiKey,
			"label":  label,
		},
	})

	if err != nil {
		log.Fatalf("An error occurred while serializing request body %v", err)
	}

	resp, err := http.Post(host+"/v1/reports.create", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalln("Request failed with:", string(body))
	}

	var parsed CreateReportResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		log.Println("Can not unmarshal JSON")
	}

	return parsed
}

func PublishDataPoints(host string, reportId string, reportToken string, dataPoints []MetricDataPoint, dataPointsByLabel map[string][]MetricDataPoint) {
	batch := 200
	for i := 0; i < len(dataPoints); i += batch {
		j := i + batch
		if j > len(dataPoints) {
			j = len(dataPoints)
		}

		dpByLabelBatch := make(map[string][]MetricDataPoint)
		for k, v := range dataPointsByLabel {
			if i < len(v) && j > len(v) {
				dpByLabelBatch[k] = v[i:j]
			}
		}

		PublishDataPointsBatch(host, reportId, reportToken, dataPoints[i:j], dpByLabelBatch)
	}
}

func PublishDataPointsBatch(host string, reportId string, reportToken string, dataPoints []MetricDataPoint, dataPointsByLabel map[string][]MetricDataPoint) {
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
		log.Fatalf("An error occurred while serializing request body %v", err)
	}

	resp, err := http.Post(host+"/v1/reports.updateData", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalln("Request failed with:", string(body))
	}
}

func PublishMetricSummary(host string, reportId string, reportToken string, metrics MetricSummary, metricsByLabel map[string]MetricSummary) {
	postBody, err := json.Marshal(PublishMetricSummaryRequest{
		Data: PublishMetricSummaryRequestData{
			ReportUUID:     reportId,
			Token:          reportToken,
			Metrics:        metrics,
			MetricsByLabel: metricsByLabel,
		},
	})

	if err != nil {
		log.Fatalf("An error occurred while serializing request body %v", err)
	}

	resp, err := http.Post(host+"/v1/reports.update", "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalln("Request failed with:", string(body))
	}
}
