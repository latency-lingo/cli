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
		result := ParseTimeStamp(format)
		if int64(ParseTimeStamp(format)) != currentTime.Unix() {
			t.Error("Failed to parse timeStamp: ", format, " result: ", result, " expected: ", currentTime.Unix())
		}
	}
}
