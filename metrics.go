package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// prometheusClient is capable of retrieving a scalar metric value from a Prometheus server.
type prometheusClient struct {
	url string
	key string
	c   http.Client
}

// Get returns the current metric data.
func (p *prometheusClient) Get() (int64, error) {
	resp, err := p.c.Get(p.url)
	if err != nil {
		return 0, err
	}
	return extractMetricsData(resp.Body, p.key)
}

// NewPrometheusClient returns a client that can fetch a specific metric value from a server.
func NewPrometheusClient(url, key string) *prometheusClient {
	return &prometheusClient{
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
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		s := sc.Text()
		if strings.HasPrefix(s, "#") {
			continue
		}
		if !strings.HasPrefix(s, keyAndSpace) {
			continue
		}
		s = strings.TrimPrefix(s, keyAndSpace)
		s = strings.TrimSpace(s)
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
	return result, nil
}
