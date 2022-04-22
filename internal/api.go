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

	//Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var parsed CreateReportResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		log.Println("Can not unmarshal JSON")
	}

	return parsed
}
