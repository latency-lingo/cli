package internal

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"sort"
)

const MaxFileSize = 1000 * 1000 * 100 // 100MB

func ParseDataFile(file string) []UngroupedMetricDataPoint {
	var (
		rows []UngroupedMetricDataPoint
	)

	validateFile(file)

	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	csvReader := csv.NewReader(f)

	// skip header
	if _, err := csvReader.Read(); err != nil {
		log.Fatal(err)
		panic(err)
	}

	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		rows = append(rows, TranslateJmeterRow((rec)))
	}

	sort.SliceStable(rows, func(i int, j int) bool {
		return rows[i].TimeStamp < rows[j].TimeStamp
	})

	return rows
}

func validateFile(file string) {
	info, err := os.Stat(file)
	if os.IsNotExist(err) {
		log.Fatalln("File", file, "does not exist")
		return
	}

	if info.Size() > MaxFileSize {
		log.Fatalln("File", file, "is too large. There is currently a 100MB limit, but please reach out with your use case.")
	}
}
