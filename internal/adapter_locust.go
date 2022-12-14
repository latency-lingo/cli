package internal

import (
	"errors"
	"log"
	"strconv"
	"strings"
)

func buildDefaultColumnIndicesLocust() map[string]int {
	// Timestamp,User Count,Type,Name,Requests/s,Failures/s,50%,66%,75%,80%,90%,95%,98%,99%,99.9%,99.99%,100%,Total Request Count,Total Failure Count,Total Median Response Time,Total Average Response Time,Total Min Response Time,Total Max Response Time,Total Average Content Size
	// 1647453612,1,GET,/v1/simulations/latency?level=low,0.000000,0.000000,0,0,0,0,0,0,0,0,0,0,0,1,0,41.29773599999997,41.29773599999997,41.29773599999997,41.29773599999997,16.0
	return map[string]int{
		"Timestamp":                   -1,
		"User Count":                  -1,
		"Type":                        -1,
		"Name":                        -1,
		"Total Request Count":         -1,
		"Total Failure Count":         -1,
		"Total Average Response Time": -1,
	}
}

func BuildColumnIndicesLocust(row []string) (map[string]int, error) {
	indices := buildDefaultColumnIndicesLocust()
	for i, header := range row {
		if _, ok := indices[header]; ok {
			indices[header] = i
		}
	}

	missing := []string{}

	for column, index := range indices {
		if index == -1 {
			missing = append(missing, column)
		}
	}

	if len(missing) > 0 {
		return nil, errors.New("missing column(s): " + strings.Join(missing, ", "))
	}

	return indices, nil
}

func TranslateLocustRow(row []string, indices map[string]int) UngroupedMetricDataPoint {
	var (
		requests     uint64
		failures     uint64
		virtualUsers uint64
		latency      float64
	)

	requests, err := strconv.ParseUint(row[indices["Total Request Count"]], 10, 64)
	if err != nil {
		log.Fatalf("failed to parse requests: %v", err)
	}

	if requests > 1 {
		log.Fatalf("requests > 1: %v", requests)
	}

	failures, err = strconv.ParseUint(row[indices["Total Failure Count"]], 10, 64)
	if err != nil {
		log.Fatalf("failed to parse failures: %v", err)
	}

	virtualUsers, err = strconv.ParseUint(row[indices["User Count"]], 10, 64)
	if err != nil {
		log.Fatalf("failed to parse virtual users: %v", err)
	}

	latency, err = strconv.ParseFloat(row[indices["Total Average Response Time"]], 64)
	if err != nil {
		log.Fatalf("failed to parse latency: %v", err)
	}

	return UngroupedMetricDataPoint{
		Requests:     requests,
		Failures:     failures,
		VirtualUsers: virtualUsers,
		TimeStamp:    ParseTimeStampMillis(row[indices["Timestamp"]]) / 1000,
		Latency:      uint64(latency),
		Label:        row[indices["Name"]],
	}
}
