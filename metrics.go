package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type promotheusClient struct {
	url string
	key string
	c   http.Client
}

func (p *promotheusClient) Get() (int64, error) {
	resp, err := p.c.Get(p.url)
	if err != nil {
		return 0, err
	}
	return extractMetricsData(resp.Body, p.key)
}

func NewPromotheusClient(url, key string) *promotheusClient {
	return &promotheusClient{
		url: url,
		key: key,
		c:   http.Client{},
	}
}

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
