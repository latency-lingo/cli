package internal

import (
	"errors"
	"log"
	"strconv"
	"strings"
)

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

func buildDefaultColumnIndices() map[string]int {
	// timeStamp,elapsed,label,responseCode,responseMessage,threadName,dataType,success,failureMessage,bytes,sentBytes,grpThreads,allThreads,URL,Latency,IdleTime,Connect
	// TODO(bobsin): add support for Latency
	return map[string]int{
		"timeStamp":       -1,
		"elapsed":         -1,
		"label":           -1,
		"responseCode":    -1,
		"responseMessage": -1,
		"threadName":      -1,
		"dataType":        -1,
		"success":         -1,
		"failureMessage":  -1,
		"bytes":           -1,
		"sentBytes":       -1,
		"grpThreads":      -1,
		"allThreads":      -1,
		"URL":             -1,
		"IdleTime":        -1,
		"Connect":         -1,
	}
}

func BuildColumnIndicesV2(row []string) (map[string]int, error) {
	columnIndices := buildDefaultColumnIndices()
	for i, column := range row {
		// TODO(bobsin): handle case where unknown column
		columnIndices[column] = i
	}

	missing := []string{}

	for column, index := range columnIndices {
		if index == -1 {
			missing = append(missing, column)
		}
	}

	if len(missing) > 0 {
		return nil, errors.New("missing column(s): " + strings.Join(missing, ", "))
	}

	return columnIndices, nil
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

func TranslateJmeterRowSample(row []string, indices map[string]int) LingoSample {
	// timeStamp,elapsed,label,responseCode,responseMessage,threadName,dataType,success,failureMessage,bytes,sentBytes,grpThreads,allThreads,URL,Latency,IdleTime,Connect
	allThreads, err := strconv.ParseInt(row[indices["allThreads"]], 10, 64)
	if err != nil {
		log.Println("error parsing allThreads", err)
	}

	elapsed, err := strconv.ParseUint(row[indices["elapsed"]], 10, 64)
	if err != nil {
		log.Println("error parsing elapsed", err)
	}

	responseCode, err := strconv.ParseInt(row[indices["responseCode"]], 10, 64)
	if err != nil {
		log.Println("error parsing responseCode", err)
	}

	bytes, err := strconv.ParseInt(row[indices["bytes"]], 10, 64)
	if err != nil {
		log.Println("error parsing bytes", err)
	}

	sentBytes, err := strconv.ParseInt(row[indices["sentBytes"]], 10, 64)
	if err != nil {
		log.Println("error parsing sentBytes", err)
	}

	grpThreads, err := strconv.ParseInt(row[indices["grpThreads"]], 10, 64)
	if err != nil {
		log.Println("error parsing grpThreads", err)
	}

	idleTime, err := strconv.ParseUint(row[indices["IdleTime"]], 10, 64)
	if err != nil {
		log.Println("error parsing IdleTime", err)
	}

	connect, err := strconv.ParseUint(row[indices["Connect"]], 10, 64)
	if err != nil {
		log.Println("error parsing Connect", err)
	}

	return LingoSample{
		Success:         row[indices["success"]] == "true",
		Elapsed:         elapsed,
		TimeStamp:       ParseTimeStampMillis(row[indices["timeStamp"]]),
		Label:           row[indices["label"]],
		ResponseCode:    int(responseCode),
		ResponseMessage: row[indices["responseMessage"]],
		ThreadName:      row[indices["threadName"]],
		DataType:        row[indices["dataType"]],
		FailureMessage:  row[indices["failureMessage"]],
		Bytes:           int(bytes),
		SentBytes:       int(sentBytes),
		GrpThreads:      int(grpThreads),
		AllThreads:      int(allThreads),
		URL:             row[indices["URL"]],
		IdleTime:        idleTime,
		Connect:         connect,
	}
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
		TimeStamp:    ParseTimeStampMillis(row[indices.TimeStamp]) / 1000,
		Latency:      latency,
	}

	return parsed
}
