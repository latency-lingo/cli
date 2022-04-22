/*
Copyright © 2022 Anthony Bobsin anthony.bobsin.dev@gmail.com

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

type GlobalDataCounter struct {
	TotalRequests   uint64
	TotalFailures   uint64
	MaxVirtualUsers uint64
	RawLatencies    []float64
}

var (
	dataFile          string
	reportUuid        string
	environment       string
	globalDataCounter GlobalDataCounter
)

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Command to publish result datasets as a Latency Lingo performance test report.",
	Long: `Command to create a performance test report on Latency Lingo based on the specified
test results dataset.`,
	Run: func(cmd *cobra.Command, args []string) {
		if environment != "production" && environment != "development" {
			log.Fatalln("User specified unknown environment", environment)
		}

		log.Println("Parsing provided file", dataFile)

		if reportUuid == "" {
			reportUuid = internal.CreateReport(hostName(environment), "Test Report").Result.Data.ID
			log.Println("Created a new report")
		}

		log.Println("Using report", reportUuid)

		rows := parseDataFile(dataFile)
		dataPoints := groupDataPoints(rows)

		internal.PublishDataPoints(hostName(environment), reportUuid, dataPoints)
		log.Println("Published", len(dataPoints), "data points")

		metricSummary := calculateMetricSummary(globalDataCounter)
		internal.PublishMetricSummary(hostName(environment), reportUuid, metricSummary)
		log.Println("Published metric summary")

		switch environment {
		case "production":
			log.Printf("Report can be found at https://latencylingo.com/reports/%s", reportUuid)
		case "development":
			log.Printf("Report can be found at http://localhost:3000/reports/%s", reportUuid)
		}
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVar(&dataFile, "file", "", "Test results file to parse and publish.")
	publishCmd.Flags().StringVar(&reportUuid, "report", "", "Existing report to publish metrics for. If not provided, a new report will be created.")
	publishCmd.Flags().StringVar(&environment, "env", "production", "Environment for API communication. Supported values: development, production.")
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

	grouped.Latencies = calculateLatencySummary(latencies)

	globalDataCounter.TotalRequests += grouped.Requests
	globalDataCounter.TotalFailures += grouped.Failures
	if grouped.VirtualUsers > globalDataCounter.MaxVirtualUsers {
		globalDataCounter.MaxVirtualUsers = grouped.VirtualUsers
	}
	globalDataCounter.RawLatencies = append(globalDataCounter.RawLatencies, latencies...)

	return grouped
}

func calculateLatencySummary(latencies []float64) *internal.Latencies {
	summary := internal.Latencies{}
	summary.AvgMs, _ = stats.Mean(latencies)
	summary.MaxMs, _ = stats.Max(latencies)
	summary.MinMs, _ = stats.Min(latencies)
	summary.P50Ms, _ = stats.Percentile(latencies, 50)
	summary.P75Ms, _ = stats.Percentile(latencies, 75)
	summary.P90Ms, _ = stats.Percentile(latencies, 90)
	summary.P95Ms, _ = stats.Percentile(latencies, 95)
	summary.P99Ms, _ = stats.Percentile(latencies, 99)

	summary.AvgMs, _ = stats.Round(summary.AvgMs, 2)
	summary.MaxMs, _ = stats.Round(summary.MaxMs, 2)
	summary.MinMs, _ = stats.Round(summary.MinMs, 2)
	summary.P50Ms, _ = stats.Round(summary.P50Ms, 2)
	summary.P75Ms, _ = stats.Round(summary.P75Ms, 2)
	summary.P90Ms, _ = stats.Round(summary.P90Ms, 2)
	summary.P95Ms, _ = stats.Round(summary.P95Ms, 2)
	summary.P99Ms, _ = stats.Round(summary.P99Ms, 2)

	return &summary
}

func calculateMetricSummary(globalDataCounter GlobalDataCounter) *internal.MetricSummary {
	summary := internal.MetricSummary{}

	summary.TotalRequests = globalDataCounter.TotalRequests
	summary.TotalFailures = globalDataCounter.TotalFailures
	summary.MaxVirtualUsers = globalDataCounter.MaxVirtualUsers
	summary.Latencies = calculateLatencySummary(globalDataCounter.RawLatencies)

	return &summary
}

func calculateIntervalFloor(timeStamp uint64) uint64 {
	difference := timeStamp % 60 % 5
	return timeStamp - difference
}

func hostName(env string) string {
	switch env {
	case "production":
		return "https://latency-lingo.web.app"
	case "development":
		return "http://localhost:5000"
	default:
		log.Fatalln("User specified unknown environment", env)
		return ""
	}
}
