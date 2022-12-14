package internal

import (
	"bufio"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

func ParseDataFileGatling(file string) ([]UngroupedMetricDataPoint, error) {
	// REQUEST		GET low latency	1647457744634	1647457744825	OK

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
		line := scanner.Text()
		row := strings.Split(line, "\t")
		if row[0] == "REQUEST" {
			rows = append(rows, TranslateGatlingRow(row))
		}
	}

	sort.SliceStable(rows, func(i int, j int) bool {
		return rows[i].TimeStamp < rows[j].TimeStamp
	})

	return rows, nil
}
