/*
Copyright © 2022 Anthony Bobsin anthony.bobsin.dev@gmail.com

*/
package cmd

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/getsentry/sentry-go"
	"github.com/latency-lingo/cli/internal"
	"github.com/spf13/cobra"
)

var (
	dataFile    string
	reportLabel string
	environment string
	apiKey      string
	rawSamples  bool
	format      string
)

// PublishCmd represents the publish command
var PublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Command to publish result datasets as a Latency Lingo performance test report.",
	Long:  `Command to create a performance test report on Latency Lingo based on the specified test results dataset.`,
	Run: func(cmd *cobra.Command, args []string) {
		initSentryScope()
		span := sentry.StartSpan(context.Background(), "Run", sentry.TransactionName("publish"))
		defer span.Finish()

		if environment != "production" && environment != "development" {
			log.Fatalln("Received unknown environment", environment)
		}

		InfoLog.Println("Parsing provided file", dataFile)
		var (
			reportPath string
			runId      string
			err        error
		)

		if rawSamples {
			runId, err = publishRawSamples()
		} else {
			runId, err = publishV2()
		}

		if err != nil {
			sentry.CaptureException(err)
			log.Printf("Failed to publish: %v", err)
			os.Exit(1)
			return
		}
		reportPath = "test-runs/" + runId

		switch environment {
		case "production":
			InfoLog.Printf("Report can be found at https://latencylingo.com/%s", reportPath)
		case "development":
			InfoLog.Printf("Report can be found at http://localhost:3000/%s", reportPath)
		}
	},
}

func init() {
	PublishCmd.Flags().StringVar(&dataFile, "file", "", "Test results file to parse and publish.")
	PublishCmd.Flags().StringVar(&reportLabel, "label", "", "Test scenario name for this run.")
	PublishCmd.Flags().StringVar(&environment, "env", "production", "Environment for API communication. Supported values: development, production.")
	PublishCmd.Flags().StringVar(&apiKey, "api-key", "", "API key to associate test runs with a user. Sign up to get one at https://latencylingo.com/account/api-access")
	PublishCmd.Flags().BoolVar(&rawSamples, "all-samples", false, "Publish all samples instead of pre-aggregated metrics.")
	PublishCmd.Flags().StringVar(&format, "format", "jmeter", "Format of the provided file. Supported values: jmeter, k6, locust, gatling.")
	PublishCmd.MarkFlagRequired("file")
	PublishCmd.MarkFlagRequired("api-key")
	PublishCmd.MarkFlagRequired("label")
}

func publishRawSamples() (string, error) {
	samples, err := internal.ParseDataFileSamples(dataFile)
	if err != nil {
		return "", err
	}

	testRun, err := internal.CreateTestRun(
		hostName(environment),
		apiKey,
		reportLabel,
		samples[0].TimeStamp/1000,
		0,
		// TODO(bobsin): make this more accurate.
		"listener",
	)
	if err != nil {
		return "", err
	}
	runId := testRun.ID
	runToken := testRun.WriteToken

	InfoLog.Println("Created a new test run with ID", runId, "under scenario", testRun.ScenarioId)

	if _, err := internal.CreateTestSamples(
		hostName(environment),
		runToken,
		samples,
	); err != nil {
		return "", err
	}

	InfoLog.Println("Published", len(samples), "samples")

	if _, err := internal.UpdateTestRun(
		hostName(environment),
		runToken,
		samples[len(samples)-1].TimeStamp/1000,
	); err != nil {
		return "", err
	}

	return runId, nil
}

func publishV2() (string, error) {
	rows, err := internal.ParseDataFile(dataFile, format)
	if err != nil {
		return "", err
	}
	groupedResult := internal.GroupAllDataPoints(rows)

	testRun, err := internal.CreateTestRun(hostName(environment), apiKey, reportLabel, rows[0].TimeStamp, rows[len(rows)-1].TimeStamp, "file")
	if err != nil {
		return "", err
	}
	runId := testRun.ID
	runToken := testRun.WriteToken

	InfoLog.Println("Created a new test run with ID", runId, "under scenario", testRun.ScenarioId)

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
	InfoLog.Println("Published", len(groupedResult.DataPoints)+labeledDpCount, "chart metric rows")

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

	InfoLog.Println("Published", len(metricSummaryByLabel)+1, "summary metric rows")

	result, err := internal.GetTestRunResults(hostName(environment), runToken)
	if err != nil {
		return "", err
	}

	jsonResult, err := json.Marshal(&result)
	if err != nil {
		return "", err
	}
	InfoLog.Println("Test run status", string(jsonResult))

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
		scope.SetUser(sentry.User{
			ID: userRef,
		})
	}

	scope.SetContext("Flags", map[string]string{
		"environment": environment,
		"user":        userRef,
		"dataFile":    dataFile,
		"reportLabel": reportLabel,
		"version":     "2.0.0",
	})
}
