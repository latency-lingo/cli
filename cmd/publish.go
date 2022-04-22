/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"sort"

	"github.com/AnthonyBobsin/latency-lingo-cli/internal"
	"github.com/montanaflynn/stats"
	"github.com/spf13/cobra"
)

var (
	dataFile string
)

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Command to publish result datasets as a Latency Lingo performance test report.",
	Long: `Command to create a performance test report on Latency Lingo based on the specified
test results dataset.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("publish called with file: ", dataFile)

		reportResponse := internal.CreateReport("http://localhost:5000", "Test Report Golang")

		log.Println("Created report", reportResponse.Result.Data.ID)

		rows := parseDataFile(dataFile)
		dataPoints := groupDataPoints(rows)

		internal.PublishDataPoints("http://localhost:5000", reportResponse.Result.Data.ID, dataPoints)
		log.Println("Published", len(dataPoints), "data points")
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVar(&dataFile, "file", "", "Test results file to parse and publish")
	publishCmd.MarkFlagRequired("file")
}

func parseDataFile(file string) []internal.UngroupedMetricDataPoint {
	var (
		rows []internal.UngroupedMetricDataPoint
	)

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

		rows = append(rows, internal.TranslateJmeterRow((rec)))
	}

	sort.SliceStable(rows, func(i int, j int) bool {
		return rows[i].TimeStamp < rows[j].TimeStamp
	})

	return rows
}

func groupDataPoints(ungrouped []internal.UngroupedMetricDataPoint) []internal.MetricDataPoint {
	var (
		startTime  uint64
		dataPoints []internal.MetricDataPoint
		batch      []internal.UngroupedMetricDataPoint
	)

	for _, dp := range ungrouped {
		if startTime == 0 {
			// init start time to last 5s interval from time stamp
			startTime = calculateIntervalFloor(dp.TimeStamp)
		}

		if dp.TimeStamp-startTime > 5 {
			dataPoints = append(dataPoints, groupDataPointBatch(batch, startTime))
			batch = nil
			startTime = 0
		}

		batch = append(batch, dp)
	}

	// TODO(bobsin) use remaining in ungrouped variable
	if len(batch) > 0 {
		dataPoints = append(dataPoints, groupDataPointBatch(batch, calculateIntervalFloor(batch[0].TimeStamp)))
	}

	return dataPoints
}

func groupDataPointBatch(ungrouped []internal.UngroupedMetricDataPoint, startTime uint64) internal.MetricDataPoint {
	var (
		latencies []float64
		grouped   internal.MetricDataPoint
	)

	grouped.TimeStamp = startTime

	for _, dp := range ungrouped {
		grouped.Requests += dp.Requests
		grouped.Failures += dp.Failures

		if dp.VirtualUsers > grouped.VirtualUsers {
			grouped.VirtualUsers = dp.VirtualUsers
		}

		latencies = append(latencies, float64(dp.Latency))
	}

	// TODO(bobsin): derive globals
	grouped.Latencies = &internal.Latencies{}
	grouped.Latencies.AvgMs, _ = stats.Mean(latencies)
	grouped.Latencies.MaxMs, _ = stats.Max(latencies)
	grouped.Latencies.MinMs, _ = stats.Min(latencies)
	grouped.Latencies.P50Ms, _ = stats.Percentile(latencies, 50)
	grouped.Latencies.P75Ms, _ = stats.Percentile(latencies, 75)
	grouped.Latencies.P90Ms, _ = stats.Percentile(latencies, 90)
	grouped.Latencies.P95Ms, _ = stats.Percentile(latencies, 95)
	grouped.Latencies.P99Ms, _ = stats.Percentile(latencies, 99)

	return grouped
}

func calculateIntervalFloor(timeStamp uint64) uint64 {
	difference := timeStamp % 60 % 5
	return timeStamp - difference
}
