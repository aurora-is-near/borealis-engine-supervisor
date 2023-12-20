package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractMetricsData(t *testing.T) {
	ttable := []struct {
		name, input, key, wantErr string
		want                      int64
	}{
		{
			name: "value is extracted successfully",
			input: `
# HELP engine_http_prometheus_requests_total Total count of Prometheus requests received
# TYPE engine_http_prometheus_requests_total counter
engine_http_prometheus_requests_total 1
# HELP engine_last_block_height_processed Block height of the last message processed
# TYPE engine_last_block_height_processed gauge
engine_last_block_height_processed{version="1.3.1"}93273959
`,
			key:  "engine_last_block_height_processed",
			want: 93273959,
		},
		{
			name: "lines starting with '#' are ignored",
			input: `
# engine_last_block_height_processed 123
engine_last_block_height_processed 456
`,
			key:  "engine_last_block_height_processed",
			want: 456,
		},
		{
			name: "correct key is extracted",
			input: `
engine_last_block_height_processed 123
engine_last_block_height 456
`,
			key:  "engine_last_block_height",
			want: 456,
		},
		{
			name: "non-existent key returns error",
			input: `
engine_last_block_height_processed 123
engine_last_block_height 456
`,
			key:     "foobar-not-found",
			wantErr: "could not find datapoint",
		},
	}

	for _, tc := range ttable {
		t.Run(tc.name, func(t *testing.T) {
			res, err := extractMetricsData(strings.NewReader(tc.input), tc.key)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
			} else {
				assert.Equal(t, tc.want, res)
			}
		})
	}
}
