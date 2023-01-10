/*
Copyright © 2022 Anthony Bobsin anthony.bobsin.dev@gmail.com

*/
package cmd

import (
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "latency-lingo-cli",
	Short: "Tool to publish performance test data to Latency Lingo APIs",
	Long: `Latency Lingo is a platform to help your team analyze and collaborate on web performance test results.

This tool helps you publish test metrics from your existing load test runner to our APIs. This is required to leverage our UI.

It supports JMeter with planned support for Locust, Gatling, and k6.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

var (
	InfoLog *log.Logger
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	InfoLog = log.Default()
	InfoLog.SetOutput(os.Stdout)

	setupSentry()

	defer sentry.Flush(2 * time.Second)
	defer func() {
		if err := recover(); err != nil {
			log.Println("Unexpected error received:", err, ". Our team has been notified, but please contact support for more information.")
			sentry.CurrentHub().Recover(err)
		}
	}()

	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.latency-lingo-cli.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	RootCmd.AddCommand(PublishCmd, CompletionCmd, UpdateCmd)
}

func setupSentry() {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              "https://db842398a24b4242bfdcdc4d5d4bf85f@o1352488.ingest.sentry.io/6633890",
		TracesSampleRate: 1.0,
		AttachStacktrace: true,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
}
