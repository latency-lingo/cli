package cmd

import (
	"context"
	"log"

	"github.com/blang/semver"
	"github.com/getsentry/sentry-go"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

const version = "2.0.5"

var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the CLI to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		initSentryScope()
		span := sentry.StartSpan(context.Background(), "Run", sentry.TransactionName("publish"))
		defer span.Finish()

		v := semver.MustParse(version)
		latest, err := selfupdate.UpdateSelf(v, "latency-lingo/cli")
		if err != nil {
			log.Println("Update failed:", err)
			return
		}
		if latest.Version.Equals(v) {
			InfoLog.Println("Already up-to-date!", version)
		} else {
			InfoLog.Println("Successfully updated to version", latest.Version)
			InfoLog.Println("Release note:\n", latest.ReleaseNotes)
		}
	},
}
