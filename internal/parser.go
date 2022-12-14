package internal

import (
	"fmt"
	"os"
)

const MaxFileSize = 1000 * 1000 * 100 // 100MB

func ParseDataFile(file string, format string) ([]UngroupedMetricDataPoint, error) {
	validateFile(file)

	switch format {
	case "jmeter":
		return ParseDataFileJmeter(file)
	case "k6":
		return ParseDataFileK6(file)
	case "gatling":
		return ParseDataFileGatling(file)
	default:
		return nil, fmt.Errorf("unsupported format %s", format)
	}
}

func validateFile(file string) error {
	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", file)
	}

	if info.Size() > MaxFileSize {
		return fmt.Errorf("file %s is too large. The current limit is 100MB, but please reach out with your use case", file)
	}

	return nil
}
