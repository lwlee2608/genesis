package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func tempCache(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "update.json")
	old := cachePath
	cachePath = func() string { return path }
	t.Cleanup(func() { cachePath = old })
	return path
}

func TestNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"v0.4.0", "v0.3.2", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.3.10", "v0.3.9", true},
		{"v0.3.2", "v0.3.2", false},
		{"v0.3.1", "v0.3.2", false},
		{"v0.4.0", "dev", false},
		{"", "v0.3.2", false},
		{"v0.4", "v0.3.2", false},
	}
	for _, tt := range tests {
		if got := newer(tt.latest, tt.current); got != tt.want {
			t.Errorf("newer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestFetchLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v9.9.9"}`))
	}))
	defer srv.Close()

	got, err := fetchLatest(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v9.9.9" {
		t.Errorf("fetchLatest() = %q, want v9.9.9", got)
	}
}

func TestFetchLatest_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	if _, err := fetchLatest(context.Background(), srv.URL); err == nil {
		t.Error("expected error on 403")
	}
}

func TestCheck_SkipsDev(t *testing.T) {
	tempCache(t)
	notice, wait := Check("dev")
	wait()
	if notice != "" {
		t.Errorf("Check(dev) = %q, want empty", notice)
	}
}

func TestCheck_NoticeFromFreshCache(t *testing.T) {
	tempCache(t)
	store(entry{Latest: "v0.4.0", CheckedAt: time.Now().Unix()})

	notice, wait := Check("v0.3.2")
	wait()

	want := "Update available: v0.3.2 → v0.4.0\n  curl -fsSL https://raw.githubusercontent.com/lwlee2608/genesis/main/scripts/install.sh | bash"
	if notice != want {
		t.Errorf("Check() = %q, want %q", notice, want)
	}
}

func TestCheck_NoNoticeWhenCurrent(t *testing.T) {
	tempCache(t)
	store(entry{Latest: "v0.3.2", CheckedAt: time.Now().Unix()})

	notice, wait := Check("v0.3.2")
	wait()
	if notice != "" {
		t.Errorf("Check() = %q, want empty when up to date", notice)
	}
}

func TestCheck_SkipsWhenDisabled(t *testing.T) {
	tempCache(t)
	t.Setenv("GENESIS_NO_UPDATE_CHECK", "1")

	notice, wait := Check("v0.3.2")
	wait()
	if notice != "" {
		t.Errorf("Check() = %q, want empty when disabled", notice)
	}
}

func TestCacheRoundTrip(t *testing.T) {
	tempCache(t)
	want := entry{Latest: "v1.2.3", CheckedAt: 1700000000}
	store(want)
	if got := load(); got != want {
		t.Errorf("load() = %+v, want %+v", got, want)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	tempCache(t)
	if got := load(); got != (entry{}) {
		t.Errorf("load() = %+v, want zero entry", got)
	}
}
