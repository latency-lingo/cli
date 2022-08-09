/*
Copyright © 2022 Anthony Bobsin anthony.bobsin.dev@gmail.com

*/
package cmd

import (
	"log"

	"github.com/AnthonyBobsin/latency-lingo-cli/internal"
	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
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

		if version == "v2" {
			if apiKey == "" {
				log.Fatalln("API key is required for version 2. Please sign up and provide an API key using the --api-key flag.")
			}

			runId, err := publishV2()
			if err != nil {
				sentry.CaptureException(err)
				log.Printf("Error when publishing test run: %v", err)
				return
			}
			reportPath = "test-runs/" + runId
		} else if version == "v1" {
			reportId, err := publishV1()
			if err != nil {
				sentry.CaptureException(err)
				log.Printf("Error when publishing report: %v", err)
				return
			}
			reportPath = "reports/" + reportId
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

func publishV1() (string, error) {
	rows, err := internal.ParseDataFile(dataFile)
	if err != nil {
		return "", errors.Errorf("error when parsing data-file: %w", err)
	}

	groupedResult := internal.GroupDataPoints(rows, internal.FiveSeconds)

	var reportToken string
	if reportUuid == "" {
		if response, err := internal.CreateReport(hostName(environment), apiKey, reportLabel); err != nil {
			return "", err
		} else {
			reportUuid = response.Result.Data.ID
			reportToken = response.Result.Data.WriteToken
		}
		log.Println("Created a new report")
	}

	log.Println("Using report", reportUuid)

	if _, err := internal.PublishDataPoints(
		hostName(environment),
		reportUuid,
		reportToken,
		groupedResult.DataPoints,
		groupedResult.DataPointsByLabel,
	); err != nil {
		return "", err
	}

	labeledDpCount := 0
	for _, dp := range groupedResult.DataPointsByLabel {
		labeledDpCount += len(dp)
	}
	log.Println("published", len(groupedResult.DataPoints)+labeledDpCount, "data points")

	metricSummary := internal.CalculateMetricSummaryOverall()
	metricSummaryByLabel := internal.CalculateMetricSummaryByLabel()

	if _, err := internal.PublishMetricSummary(
		hostName(environment),
		reportUuid,
		reportToken,
		metricSummary,
		metricSummaryByLabel,
	); err != nil {
		return "", err
	}

	log.Println("Published", len(metricSummaryByLabel)+1, "metric summary rows")

	return reportUuid, nil
}

func publishV2() (string, error) {
	rows, err := internal.ParseDataFile(dataFile)
	if err != nil {
		return "", errors.Errorf("error when parsing data-file: %w", err)
	}
	groupedResult := internal.GroupAllDataPoints(rows)

	testRun, err := internal.CreateTestRun(hostName(environment), apiKey, reportLabel, rows[0].TimeStamp, rows[len(rows)-1].TimeStamp)
	if err != nil {
		return "", err
	}
	runId := testRun.ID
	runToken := testRun.WriteToken

	log.Println("Created a new test run with ID", runId)

	if _, err := internal.CreateTestChartMetrics(
		hostName(environment),
		runToken,
		groupedResult.DataPoints,
		groupedResult.DataPointsByLabel,
	); err != nil {
		return "", err
	}

	labeledDpCount := 0
	for _, dp := range groupedResult.DataPointsByLabel {
		labeledDpCount += len(dp)
	}
	log.Println("Published", len(groupedResult.DataPoints)+labeledDpCount, "chart metric rows")

	metricSummary := internal.CalculateMetricSummaryOverall()
	metricSummaryByLabel := internal.CalculateMetricSummaryByLabel()

	if _, err := internal.CreateTestSummaryMetrics(
		hostName(environment),
		runToken,
		metricSummary,
		metricSummaryByLabel,
	); err != nil {
		return "", err
	}

	log.Println("Published", len(metricSummaryByLabel)+1, "summary metric rows")

	return runId, nil
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

	var userRef string
	if apiKey != "" {
		userRef = apiKey[:6] + "..." + apiKey[len(apiKey)-6:]
	}

	// TODO(bobsin): add CLI version here
	scope.SetContext("Flags", map[string]string{
		"environment": environment,
		"user":        userRef,
		"dataFile":    dataFile,
		"reportLabel": reportLabel,
	})
}
