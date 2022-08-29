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
	environment string
	apiKey      string
)

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Command to publish result datasets as a Latency Lingo performance test report.",
	Long:  `Command to create a performance test report on Latency Lingo based on the specified test results dataset.`,
	Run: func(cmd *cobra.Command, args []string) {
		initSentryScope()

		if environment != "production" && environment != "development" {
			log.Fatalln("Received unknown environment", environment)
		}

		log.Println("Parsing provided file", dataFile)
		var reportPath string

		runId, err := publishV2()
		if err != nil {
			sentry.CaptureException(err)
			log.Printf("Failed to publish: %v", err)
			return
		}
		reportPath = "test-runs/" + runId

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
	publishCmd.Flags().StringVar(&reportLabel, "label", "", "Test scenario name for this run.")
	publishCmd.Flags().StringVar(&environment, "env", "production", "Environment for API communication. Supported values: development, production.")
	publishCmd.Flags().StringVar(&apiKey, "api-key", "", "API key to associate test runs with a user. Sign up to get one at https://latencylingo.com/account/api-access")
	publishCmd.MarkFlagRequired("file")
	publishCmd.MarkFlagRequired("api-key")
	publishCmd.MarkFlagRequired("label")
}

func publishV2() (string, error) {
	rows, err := internal.ParseDataFile(dataFile)
	if err != nil {
		return "", err
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
		scope.SetUser(sentry.User{
			ID: userRef,
		})
	}

	scope.SetContext("Flags", map[string]string{
		"environment": environment,
		"user":        userRef,
		"dataFile":    dataFile,
		"reportLabel": reportLabel,
		"version":     "1.3.1",
	})
}
