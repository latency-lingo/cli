package internal

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/pkg/errors"
)

const MaxFileSize = 1000 * 1000 * 100 // 100MB

func ParseDataFile(file string) ([]UngroupedMetricDataPoint, error) {
	var (
		rows []UngroupedMetricDataPoint
	)

	if err := validateFile(file); err != nil {
		return nil, err
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, errors.Errorf("cannot open file %s: %w", file, err)
	}

	defer f.Close()

	csvReader := csv.NewReader(f)
	if _, err := csvReader.Read(); err != nil {
		// TODO(bobsin): validate default header config
		return nil, errors.Errorf("cannot read file %s: %w", file, err)
	}

	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, errors.Errorf("cannot read file %s: %w", file, err)
		}
		rows = append(rows, TranslateJmeterRow(rec))
	}

	sort.SliceStable(rows, func(i int, j int) bool {
		return rows[i].TimeStamp < rows[j].TimeStamp
	})

	return rows, nil
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
