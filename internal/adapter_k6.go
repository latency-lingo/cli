package internal

type K6MetricTags struct {
	ExpectedResponse string `json:"expected_response"`
	Group            string `json:"group"`
	Method           string `json:"method"`
	Name             string `json:"name"`
	Proto            string `json:"proto"`
	Scenario         string `json:"scenario"`
	Status           string `json:"status"`
	URL              string `json:"url"`
}

type K6MetricData struct {
	Time  string       `json:"time"`
	Value float64      `json:"value"`
	Tags  K6MetricTags `json:"tags"`
}

type K6Metric struct {
	Type   string       `json:"type"`
	Data   K6MetricData `json:"data"`
	Metric string       `json:"metric"`
}

func TranslateK6Row(row K6Metric) UngroupedMetricDataPoint {
	failures := 0
	if row.Data.Tags.ExpectedResponse != "true" {
		failures = 1
	}

	return UngroupedMetricDataPoint{
		Requests:     1,
		Failures:     uint64(failures),
		VirtualUsers: 0,
		TimeStamp:    ParseTimeStampMillis(row.Data.Time) / 1000,
		Latency:      uint64(row.Data.Value),
		Label:        row.Data.Tags.Name,
	}
}
