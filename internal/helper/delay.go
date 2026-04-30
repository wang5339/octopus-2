package helper

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const urlDelayTimeout = 5 * time.Second

func GetUrlDelay(httpClient *http.Client, url string, ctx context.Context) (int, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return 0, fmt.Errorf("url is empty")
	}
	if httpClient == nil {
		return 0, fmt.Errorf("http client is nil")
	}

	delayCtx, cancel := context.WithTimeout(ctx, urlDelayTimeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(delayCtx, http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return int(time.Since(start).Milliseconds()), nil
}
