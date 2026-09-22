package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
	if _, ok := <-Check("dev"); ok {
		t.Error("expected closed channel for dev build")
	}
}

func TestNotice_UpdateAvailable(t *testing.T) {
	ch := make(chan string, 1)
	ch <- "v0.4.0"
	close(ch)

	want := "Update available: v0.3.2 → v0.4.0\n  curl -fsSL https://raw.githubusercontent.com/lwlee2608/genesis/main/scripts/install.sh | bash"
	if got := Notice("v0.3.2", ch); got != want {
		t.Errorf("Notice() = %q, want %q", got, want)
	}
}

func TestNotice_Empty(t *testing.T) {
	ch := make(chan string)
	if got := Notice("v0.3.2", ch); got != "" {
		t.Errorf("Notice() = %q, want empty when no result ready", got)
	}
}
