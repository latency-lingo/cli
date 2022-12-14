package internal

func TranslateGatlingRow(row []string) UngroupedMetricDataPoint {
	// row[0] = REQUEST
	// row[1] = ""
	// row[2] = GET low latency
	// row[3] = 1647457744634
	// row[4] = 1647457744825
	// row[5] = OK

	failures := 0
	if row[5] != "OK" {
		failures = 1
	}

	return UngroupedMetricDataPoint{
		Requests:     1,
		Failures:     uint64(failures),
		TimeStamp:    ParseTimeStampMillis(row[3]) / 1000,
		Latency:      uint64(ParseTimeStampMillis(row[4]) - ParseTimeStampMillis(row[3])),
		Label:        row[2],
		VirtualUsers: 0,
	}
}
