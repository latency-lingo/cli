package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

const MaxFileSize = 1000 * 1000 * 100 // 100MB

func ParseDataFile(file string) ([]UngroupedMetricDataPoint, error) {
	span := sentry.StartSpan(context.Background(), "ParseDataFile")
	defer span.Finish()

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

	csvReader := csv.NewReader(f)
	header, err := csvReader.Read()
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read file %s", file)
	}

	var indices *ColumnIndices
	indices, err = BuildColumnIndices(header)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot parse file %s", file)
	}

	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, errors.Wrapf(err, "cannot read file %s", file)
		}
		rows = append(rows, TranslateJmeterRow(rec, indices))
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
