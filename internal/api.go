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
			ID      string `json:"id"`
			Label   string `json:"label"`
			Metrics struct {
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
	ReportUUID string            `json:"reportUuid"`
	Action     string            `json:"action"`
	DataPoints []MetricDataPoint `json:"dataPoints"`
}

type PublishDataPointsRequest struct {
	Data *PublishDataPointsRequestData `json:"data"`
}

func CreateReport(host string, label string) CreateReportResponse {
	postBody, _ := json.Marshal(map[string]map[string]string{
		"data": {
			"label": label,
		},
	})

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

func PublishDataPoints(host string, reportId string, dataPoints []MetricDataPoint) {
	postBody, _ := json.Marshal(PublishDataPointsRequest{
		Data: &PublishDataPointsRequestData{
			ReportUUID: reportId,
			Action:     "add",
			DataPoints: dataPoints,
		},
	})

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
