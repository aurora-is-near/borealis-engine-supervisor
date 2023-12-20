package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// PrometheusClient is capable of retrieving a scalar metric value from a Prometheus server.
type PrometheusClient struct {
	url string
	key string
	c   http.Client
}

// Get returns the current metric data.
func (p *PrometheusClient) Get() (int64, error) {
	resp, err := p.c.Get(p.url)
	if err != nil {
		return 0, err
	}
	return extractMetricsData(resp.Body, p.key)
}

// NewPrometheusClient returns a client that can fetch a specific metric value from a server.
func NewPrometheusClient(url, key string) *PrometheusClient {
	return &PrometheusClient{
		url: url,
		key: key,
		c:   http.Client{},
	}
}

// extractMetricsData returns the integer value of key in the input.
// Input is lines containing the key, a space, and the value.
func extractMetricsData(r io.Reader, key string) (int64, error) {
	var (
		result int64
		found  bool
		err    error
	)
	keyAndSpace := fmt.Sprintf("%s ", key)
	keyAndVersion := fmt.Sprintf("%s{", key)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		s := sc.Text()
		if strings.HasPrefix(s, "#") {
			continue
		}
		if !strings.HasPrefix(s, keyAndSpace) && !strings.HasPrefix(s, keyAndVersion) {
			continue
		}
		s = strings.TrimSpace(strings.TrimPrefix(s, key))
		if strings.HasPrefix(s, "{") {
			p := strings.LastIndex(s, "}")
			if p >= 0 && p < len(s)-1 {
				s = strings.TrimSpace(s[p+1:])
			} else {
				continue
			}
		}
		result, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, err
		}
		found = true
		break
	}
	if !found {
		return 0, fmt.Errorf("could not find datapoint %q in metrics data", key)
	}
	_, _ = fmt.Fprintf(os.Stderr, "ProcessedBlock: %d\n", result)
	return result, nil
}
