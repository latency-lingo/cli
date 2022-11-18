package internal

import (
	"strconv"
	"testing"
	"time"
)

func TestParseTimeStamp(t *testing.T) {
	currentTime := time.Now()
	validFormats := []string{
		strconv.Itoa(int(currentTime.UnixMilli())),
		strconv.Itoa(int(currentTime.Unix())),
		currentTime.Format("2006-01-02 15:04:05"),
		currentTime.Format("2006-01-02 15:04:05.999"),
		currentTime.Format("2006/01/02 15:04:05"),
		currentTime.Format("2006/01/02 15:04:05.999"),
		currentTime.Format("2006/01/02 15:04:05.999"),
		currentTime.Format("2006-01-02 15:04:05"),
		currentTime.Format("2006-01-02T15:04:05.999Z07:00"),
	}

	for _, format := range validFormats {
		result := ParseTimeStampMillis(format)
		if int64(ParseTimeStampMillis(format)/1000) != currentTime.Unix() {
			t.Error("Failed to parse timeStamp: ", format, " result: ", result, " expected: ", currentTime.Unix())
		}
	}
}

var (
	sampleHeaders = []string{
		"timeStamp",
		"label",
		"elapsed",
		"responseCode",
		"responseMessage",
		"threadName",
		"dataType",
		"success",
		"failureMessage",
		"bytes",
		"sentBytes",
		"grpThreads",
		"allThreads",
		"URL",
		"Latency",
		"IdleTime",
		"Connect",
	}
	sampleRow = []string{
		"1610000000",
		"test",
		"100",
		"200",
		"OK",
		"thread",
		"text",
		"true",
		"",
		"100",
		"100",
		"1",
		"1",
		"http://localhost:8080",
		"100",
		"100",
		"100",
	}
)

func TestBuildColumnIndicesV2(t *testing.T) {
	indices, err := BuildColumnIndicesV2(sampleHeaders)
	if err != nil {
		t.Error("Failed to build column indices: ", err)
	}

	for i, header := range sampleHeaders {
		if indices[header] != i {
			t.Error("Failed to build column indices: ", header, " expected: ", i, " got: ", indices[header])
		}
	}
}

func TestTranslateJmeterRowSample(t *testing.T) {
	indices, err := BuildColumnIndicesV2(sampleHeaders)
	if err != nil {
		t.Error("Failed to build column indices: ", err)
	}

	row := TranslateJmeterRowSample(sampleRow, indices)

	if row.TimeStamp != 1610000000000 {
		t.Error("Failed to parse timestamp: ", row.TimeStamp, " expected: ", 1610000000000)
	}

	if row.Label != "test" {
		t.Error("Failed to parse label: ", row.Label, " expected: ", "test")
	}

	if row.Elapsed != 100 {
		t.Error("Failed to parse elapsed: ", row.Elapsed, " expected: ", 100)
	}

	if row.ResponseCode != 200 {
		t.Error("Failed to parse responseCode: ", row.ResponseCode, " expected: ", 200)
	}

	if row.ResponseMessage != "OK" {
		t.Error("Failed to parse responseMessage: ", row.ResponseMessage, " expected: ", "OK")
	}

	if row.ThreadName != "thread" {
		t.Error("Failed to parse threadName: ", row.ThreadName, " expected: ", "thread")
	}

	if row.DataType != "text" {
		t.Error("Failed to parse dataType: ", row.DataType, " expected: ", "text")
	}

	if row.Success != true {
		t.Error("Failed to parse success: ", row.Success, " expected: ", true)
	}

	if row.FailureMessage != "" {
		t.Error("Failed to parse failureMessage: ", row.FailureMessage, " expected: ", "")
	}

	if row.Bytes != 100 {
		t.Error("Failed to parse bytes: ", row.Bytes, " expected: ", 100)
	}

	if row.SentBytes != 100 {
		t.Error("Failed to parse sentBytes: ", row.SentBytes, " expected: ", 100)
	}

	if row.GrpThreads != 1 {
		t.Error("Failed to parse grpThreads: ", row.GrpThreads, " expected: ", 1)
	}

	if row.AllThreads != 1 {
		t.Error("Failed to parse allThreads: ", row.AllThreads, " expected: ", 1)
	}

	if row.URL != "http://localhost:8080" {
		t.Error("Failed to parse URL: ", row.URL, " expected: ", "http://localhost:8080")
	}

	if row.IdleTime != 100 {
		t.Error("Failed to parse IdleTime: ", row.IdleTime, " expected: ", 100)
	}

	if row.Connect != 100 {
		t.Error("Failed to parse Connect: ", row.Connect, " expected: ", 100)
	}
}
