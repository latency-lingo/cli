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

func ParseDataFileJmeter(file string) ([]UngroupedMetricDataPoint, error) {
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

func ParseDataFileSamples(file string) ([]LingoSample, error) {
	span := sentry.StartSpan(context.Background(), "ParseDataFileSamples")
	defer span.Finish()

	var (
		samples []LingoSample
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

	indices, err := BuildColumnIndicesV2(header)
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
		samples = append(samples, TranslateJmeterRowSample(rec, indices))
	}

	sort.SliceStable(samples, func(i int, j int) bool {
		return samples[i].TimeStamp < samples[j].TimeStamp
	})

	return samples, nil
}
