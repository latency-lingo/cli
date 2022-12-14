package internal

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"sort"

	"github.com/pkg/errors"
)

func ParseDataFileK6(file string) ([]UngroupedMetricDataPoint, error) {
	var (
		rows []UngroupedMetricDataPoint
	)

	if err := validateFile(file); err != nil {
		return nil, err
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot open file %s", file)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var metric K6Metric
		line := scanner.Bytes()
		if err := json.Unmarshal(line, &metric); err != nil {
			return nil, errors.Wrapf(err, "cannot parse line %s", line)
		}

		if metric.Type == "Point" && metric.Metric == "http_req_duration" {
			rows = append(rows, TranslateK6Row(metric))
		}
	}

	// Check for errors during scanning.
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		return nil, errors.Wrapf(err, "cannot read file %s", file)
	}

	sort.SliceStable(rows, func(i int, j int) bool {
		return rows[i].TimeStamp < rows[j].TimeStamp
	})

	return rows, nil
}
