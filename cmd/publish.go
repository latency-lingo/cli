/*
Copyright © 2022 Anthony Bobsin anthony.bobsin.dev@gmail.com

*/
package cmd

import (
	"log"

	"github.com/AnthonyBobsin/latency-lingo-cli/internal"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
)

var (
	dataFile    string
	reportLabel string
	reportUuid  string
	environment string
	apiKey      string
	version     string
)

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
	rows := internal.ParseDataFile(dataFile)
	groupedResult := internal.GroupDataPoints(rows, internal.FiveSeconds)

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

	metricSummary := internal.CalculateMetricSummaryOverall()
	metricSummaryByLabel := internal.CalculateMetricSummaryByLabel()

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
	rows := internal.ParseDataFile(dataFile)
	groupedResult := internal.GroupAllDataPoints(rows)

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

	metricSummary := internal.CalculateMetricSummaryOverall()
	metricSummaryByLabel := internal.CalculateMetricSummaryByLabel()

	internal.CreateTestSummaryMetrics(
		hostName(environment),
		runToken,
		metricSummary,
		metricSummaryByLabel,
	)

	log.Println("Published metric summary")

	return runId
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
