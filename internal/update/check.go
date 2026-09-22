package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	releaseURL = "https://api.github.com/repos/lwlee2608/genesis/releases/latest"
	timeout    = 2 * time.Second
)

func Check(current string) <-chan string {
	ch := make(chan string, 1)
	if current == "dev" || os.Getenv("GENESIS_NO_UPDATE_CHECK") != "" {
		close(ch)
		return ch
	}
	go func() {
		defer close(ch)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		latest, err := fetchLatest(ctx, releaseURL)
		if err == nil && newer(latest, current) {
			ch <- latest
		}
	}()
	return ch
}

func Notice(current string, ch <-chan string) string {
	select {
	case latest, ok := <-ch:
		if ok {
			return fmt.Sprintf("Update available: %s → %s\n  curl -fsSL https://raw.githubusercontent.com/lwlee2608/genesis/main/scripts/install.sh | bash", current, latest)
		}
	default:
	}
	return ""
}

func fetchLatest(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.TagName, nil
}

func newer(latest, current string) bool {
	l, lok := parse(latest)
	c, cok := parse(current)
	if !lok || !cok {
		return false
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
