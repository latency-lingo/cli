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
	"github.com/getsentry/sentry-go"
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

const MaxFileSize = 1000 * 1000 * 100 // 100MB

var (
	dataFile    string
	reportLabel string
	reportUuid  string
	environment string
	apiKey      string
	version     string
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
		initSentryScope()

		if environment != "production" && environment != "development" {
			log.Fatalln("Unknown environment", environment)
		}

		log.Println("Parsing provided file", dataFile)
		var reportPath string

		if version == "v1" {
			reportId := publishV1()
			reportPath = "reports/" + reportId
		} else if version == "v2" {
			runId := publishV2()
			reportPath = "test-runs/" + runId
		} else {
			log.Fatalln("Unknown version", version)
		}

		switch environment {
		case "production":
			log.Printf("Report can be found at https://latencylingo.com/%s", reportPath)
		case "development":
			log.Printf("Report can be found at http://localhost:3000/%s", reportPath)
		}
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVar(&dataFile, "file", "", "Test results file to parse and publish.")
	publishCmd.Flags().StringVar(&reportLabel, "label", "Test Report", "Label to use when creating a new report.")
	publishCmd.Flags().StringVar(&environment, "env", "production", "Environment for API communication. Supported values: development, production.")
	publishCmd.Flags().StringVar(&apiKey, "api-key", "", "API key to associate this report with a user.")
	publishCmd.Flags().StringVar(&version, "version", "v1", "Version of the publish command to use. Supported values: v1, v2.")
	publishCmd.MarkFlagRequired("file")
}

func publishV1() string {
	var reportToken string
	rows := parseDataFile(dataFile)
	groupedResult := groupDataPoints(rows)

	if reportUuid == "" {
		reportResponse := internal.CreateReport(
			hostName(environment),
			apiKey,
			reportLabel,
		).Result.Data
		reportUuid = reportResponse.ID
		reportToken = reportResponse.WriteToken
		log.Println("Created a new report")
	}

	log.Println("Using report", reportUuid)

	internal.PublishDataPoints(
		hostName(environment),
		reportUuid,
		reportToken,
		groupedResult.DataPoints,
		groupedResult.DataPointsByLabel,
	)
	log.Println("Published", len(groupedResult.DataPoints), "data points")

	metricSummary := calculateMetricSummary(globalDataCounter, "")
	metricSummaryByLabel := make(map[string]internal.MetricSummary)
	for label, counter := range labeledDataCounter {
		metricSummaryByLabel[label] = calculateMetricSummary(counter, label)
	}
	internal.PublishMetricSummary(
		hostName(environment),
		reportUuid,
		reportToken,
		metricSummary,
		metricSummaryByLabel,
	)
	log.Println("Published metric summary")

	return reportUuid
}

func publishV2() string {
	rows := parseDataFile(dataFile)
	groupedResult := groupDataPoints(rows)

	testRun := internal.CreateTestRun(hostName(environment), apiKey, reportLabel)
	runId := testRun.ID
	runToken := testRun.WriteToken

	log.Println("Created a new test run with ID", runId)

	internal.CreateTestChartMetrics(
		hostName(environment),
		runToken,
		groupedResult.DataPoints,
		groupedResult.DataPointsByLabel,
	)

	// TODO(bobsin): make this accurate to include labeled and time granularity
	log.Println("Published", len(groupedResult.DataPoints), "data points")

	metricSummary := calculateMetricSummary(globalDataCounter, "")
	metricSummaryByLabel := make(map[string]internal.MetricSummary)
	for label, counter := range labeledDataCounter {
		metricSummaryByLabel[label] = calculateMetricSummary(counter, label)
	}

	internal.CreateTestSummaryMetrics(
		hostName(environment),
		runToken,
		metricSummary,
		metricSummaryByLabel,
	)

	log.Println("Published metric summary")

	return runId
}

func parseDataFile(file string) []internal.UngroupedMetricDataPoint {
	var (
		rows []internal.UngroupedMetricDataPoint
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

		rows = append(rows, internal.TranslateJmeterRow((rec)))
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

func initSentryScope() {
	scope := sentry.CurrentHub().PushScope()
	scope.SetTags(map[string]string{
		"environment": environment,
	})
	// TODO(bobsin): add CLI version here
	scope.SetContext("Flags", map[string]string{
		"environment": environment,
		"user":        apiKey,
		"dataFile":    dataFile,
		"reportLabel": reportLabel,
	})
}
