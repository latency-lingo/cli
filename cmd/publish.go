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
	Label           string
	TotalRequests   uint64
	TotalFailures   uint64
	MaxVirtualUsers uint64
	RawLatencies    []float64
}

type LabeledDataCounter = map[string]*GlobalDataCounter

type GroupedResult struct {
	DataPoints        []internal.MetricDataPoint
	DataPointsByLabel map[string][]internal.MetricDataPoint
}

var (
	dataFile    string
	reportUuid  string
	reportToken string
	environment string
)

var globalDataCounter = &GlobalDataCounter{}
var labeledDataCounter = make(map[string]*GlobalDataCounter)

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
			reportResponse := internal.CreateReport(hostName(environment), "Test Report").Result.Data
			reportUuid = reportResponse.ID
			reportToken = reportResponse.WriteToken
			log.Println("Created a new report")
		}

		log.Println("Using report", reportUuid)

		rows := parseDataFile(dataFile)
		groupedResult := groupDataPoints(rows)

		internal.PublishDataPoints(hostName(environment), reportUuid, reportToken, groupedResult.DataPoints, groupedResult.DataPointsByLabel)
		log.Println("Published", len(groupedResult.DataPoints), "data points")

		metricSummary := calculateMetricSummary(globalDataCounter, "")
		metricSummaryByLabel := make(map[string]internal.MetricSummary)
		for label, counter := range labeledDataCounter {
			metricSummaryByLabel[label] = calculateMetricSummary(counter, label)
		}
		internal.PublishMetricSummary(hostName(environment), reportUuid, reportToken, metricSummary, metricSummaryByLabel)
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
	publishCmd.Flags().StringVar(&reportToken, "token", "", "Token to use when publishing metrics. Only required if `report` flag is passed.")
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

func groupDataPoints(ungrouped []internal.UngroupedMetricDataPoint) GroupedResult {
	var (
		startTime         uint64
		dataPoints        []internal.MetricDataPoint
		dataPointsByLabel map[string][]internal.MetricDataPoint
		batch             []internal.UngroupedMetricDataPoint
		batchByLabel      map[string][]internal.UngroupedMetricDataPoint
	)
	dataPointsByLabel = make(map[string][]internal.MetricDataPoint)
	batchByLabel = make(map[string][]internal.UngroupedMetricDataPoint)

	for _, dp := range ungrouped {
		if startTime == 0 {
			// init start time to last 5s interval from time stamp
			startTime = calculateIntervalFloor(dp.TimeStamp)
		}

		if dp.TimeStamp-startTime > 5 {
			dataPoints = append(dataPoints, groupDataPointBatch(batch, startTime, ""))
			mergeDataPointsByLabel(dataPointsByLabel, batchByLabel, startTime)
			batch = nil
			batchByLabel = map[string][]internal.UngroupedMetricDataPoint{}
			startTime = 0
		}

		batch = append(batch, dp)
		batchByLabel[dp.Label] = append(batchByLabel[dp.Label], dp)
	}

	if len(batch) > 0 {
		startTime = calculateIntervalFloor(batch[0].TimeStamp)
		dataPoints = append(dataPoints, groupDataPointBatch(batch, startTime, ""))
		mergeDataPointsByLabel(dataPointsByLabel, batchByLabel, startTime)
	}

	return GroupedResult{
		DataPoints:        dataPoints,
		DataPointsByLabel: dataPointsByLabel,
	}
}

func mergeDataPointsByLabel(existing map[string][]internal.MetricDataPoint, batch map[string][]internal.UngroupedMetricDataPoint, startTime uint64) {
	for label, dataPoints := range batch {
		existing[label] = append(existing[label], groupDataPointBatch(dataPoints, startTime, label))
	}
}

func groupDataPointBatch(ungrouped []internal.UngroupedMetricDataPoint, startTime uint64, label string) internal.MetricDataPoint {
	var (
		latencies []float64
		grouped   internal.MetricDataPoint
	)

	grouped.TimeStamp = startTime

	for _, dp := range ungrouped {
		if label != "" {
			grouped.Label = label
		}

		grouped.Requests += dp.Requests
		grouped.Failures += dp.Failures

		if dp.VirtualUsers > grouped.VirtualUsers {
			grouped.VirtualUsers = dp.VirtualUsers
		}

		latencies = append(latencies, float64(dp.Latency))
	}

	grouped.Latencies = calculateLatencySummary(latencies)

	if label != "" {
		updateGlobalCounter(globalDataCounter, grouped, latencies)
		if labeledDataCounter[label] == nil {
			labeledDataCounter[label] = &GlobalDataCounter{}
		}

		updateGlobalCounter(labeledDataCounter[label], grouped, latencies)
	}

	return grouped
}

func updateGlobalCounter(counter *GlobalDataCounter, metric internal.MetricDataPoint, latencies []float64) {
	counter.TotalRequests += metric.Requests
	counter.TotalFailures += metric.Failures
	if metric.VirtualUsers > counter.MaxVirtualUsers {
		counter.MaxVirtualUsers = metric.VirtualUsers
	}
	counter.RawLatencies = append(counter.RawLatencies, latencies...)
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

func calculateMetricSummary(globalDataCounter *GlobalDataCounter, label string) internal.MetricSummary {
	summary := internal.MetricSummary{}

	if label != "" {
		summary.Label = label
	}

	summary.TotalRequests = globalDataCounter.TotalRequests
	summary.TotalFailures = globalDataCounter.TotalFailures
	summary.MaxVirtualUsers = globalDataCounter.MaxVirtualUsers
	summary.Latencies = calculateLatencySummary(globalDataCounter.RawLatencies)

	return summary
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
