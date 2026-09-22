package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	releaseURL = "https://api.github.com/repos/lwlee2608/genesis/releases/latest"
	timeout    = 2 * time.Second
	ttl        = 24 * time.Hour
)

type entry struct {
	Latest    string `json:"latest"`
	CheckedAt int64  `json:"checked_at"`
}

var cachePath = defaultCachePath

func Check(current string) (string, func()) {
	noop := func() {}
	if current == "dev" || os.Getenv("GENESIS_NO_UPDATE_CHECK") != "" {
		return "", noop
	}

	cached := load()
	notice := ""
	if newer(cached.Latest, current) {
		notice = fmt.Sprintf("Update available: %s → %s\n  curl -fsSL https://raw.githubusercontent.com/lwlee2608/genesis/main/scripts/install.sh | bash", current, cached.Latest)
	}
	if time.Since(time.Unix(cached.CheckedAt, 0)) < ttl {
		return notice, noop
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		latest, err := fetchLatest(ctx, releaseURL)
		if err != nil {
			return
		}
		store(entry{Latest: latest, CheckedAt: time.Now().Unix()})
	}()

	return notice, func() {
		select {
		case <-done:
		case <-time.After(timeout):
		}
	}
}

func defaultCachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "genesis", "update.json")
}

func load() entry {
	var e entry
	path := cachePath()
	if path == "" {
		return e
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return e
	}
	json.Unmarshal(data, &e)
	return e
}

func store(e entry) {
	path := cachePath()
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0o644)
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
