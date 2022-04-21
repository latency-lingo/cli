/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/AnthonyBobsin/latency-lingo-cli/internal"
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
		fmt.Println("publish called with file: ", dataFile)

    f, err := os.Open(dataFile)
    if err != nil {
			log.Fatal(err)
    }

    defer f.Close()

    csvReader := csv.NewReader(f)

		// skip header
		if _, err := csvReader.Read(); err != nil {
			panic(err)
			log.Fatal(err)
		}

    for {
			rec, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}

			internal.TranslateJmeterRow(rec)
    }
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVar(&dataFile, "file", "", "Test results file to parse and publish")
	publishCmd.MarkFlagRequired("file");
}
