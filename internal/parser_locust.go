package internal

import (
	"context"
	"encoding/csv"
	"io"
	"os"
	"sort"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
)

func ParseDataFileLocust(file string) ([]UngroupedMetricDataPoint, error) {
	span := sentry.StartSpan(context.Background(), "ParseDataFileLocust")
	defer span.Finish()

	var (
		rows []UngroupedMetricDataPoint
	)

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

	indices, err := BuildColumnIndicesLocust(header)
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

		if rec[indices["name"]] != "Aggregated" {
			rows = append(rows, TranslateLocustRow(rec, indices))
		}
	}

	sort.SliceStable(rows, func(i int, j int) bool {
		return rows[i].TimeStamp < rows[j].TimeStamp
	})

	return rows, nil
}
