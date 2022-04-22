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
	"sort"

	"github.com/AnthonyBobsin/latency-lingo-cli/internal"
	"github.com/spf13/cobra"
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
			log.Fatal(err)
			panic(err)
		}

		var (
			startTime  uint64
			dataPoints []internal.MetricDataPoint
			ungrouped  []internal.UngroupedMetricDataPoint
		)

		for {
			rec, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}

			parsed := internal.TranslateJmeterRow(rec)
			if parsed.TimeStamp > 0 {
				ungrouped = append(ungrouped, internal.TranslateJmeterRow((rec)))
			}
		}

		sort.SliceStable(ungrouped, func(i int, j int) bool {
			return ungrouped[i].TimeStamp < ungrouped[j].TimeStamp
		})

		var batch []internal.UngroupedMetricDataPoint

		for _, dp := range ungrouped {
			if startTime == 0 {
				// init start time to last 5s interval from time stamp
				startTime = calculateIntervalFloor(dp.TimeStamp)
				log.Println("Setting startTime to: ", startTime)
			}

			if dp.TimeStamp-startTime > 5 {
				dataPoints = append(dataPoints, groupDataPointWindow(batch, startTime))
				batch = nil
				startTime = 0
			}

			batch = append(batch, dp)
		}

		// TODO(bobsin) use remaining in ungrouped variable
		if len(batch) > 0 {
			dataPoints = append(dataPoints, groupDataPointWindow(batch, calculateIntervalFloor(batch[0].TimeStamp)))
		}
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVar(&dataFile, "file", "", "Test results file to parse and publish")
	publishCmd.MarkFlagRequired("file")
}

func groupDataPointWindow(ungrouped []internal.UngroupedMetricDataPoint, startTime uint64) internal.MetricDataPoint {
	var (
		latencies []uint64
		grouped   internal.MetricDataPoint
	)

	log.Println("Grouping data points: ", ungrouped)

	grouped.TimeStamp = startTime

	for _, dp := range ungrouped {
		grouped.Requests += dp.Requests
		grouped.Failures += dp.Failures

		if dp.VirtualUsers > grouped.VirtualUsers {
			grouped.VirtualUsers = dp.VirtualUsers
		}

		latencies = append(latencies, dp.Latency)
	}

	// TODO(bobsin): properly set latencies
	// TODO(bobsin): derive globals
	grouped.Latencies = &internal.Latencies{}
	grouped.Latencies.AvgMs = calcAvg(latencies)

	log.Println("Created grouped data point: ", grouped)

	return grouped
}

func calcAvg(numbers []uint64) float32 {
	size := len(numbers)

	sum := 0
	for _, num := range numbers {
		sum += int(num)
	}
	return float32(sum / size)
}

func calculateIntervalFloor(timeStamp uint64) uint64 {
	difference := timeStamp % 60 % 5
	log.Println("Difference for timeStamp: ", difference)
	return timeStamp - difference
}
