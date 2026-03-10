package bitbucket

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestGetDiffStatFollowsPagination(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasPrefix(r.URL.String(), "/2.0/repositories/ws/repo/pullrequests/7/diffstat?pagelen=100"):
			_ = json.NewEncoder(w).Encode(PaginatedResponse[DiffStat]{
				Values: []DiffStat{{New: &DiffFile{Path: "first.go"}}},
				Next:   "http://" + r.Host + "/page-2",
			})
		case r.URL.Path == "/page-2":
			_ = json.NewEncoder(w).Encode(PaginatedResponse[DiffStat]{
				Values: []DiffStat{{New: &DiffFile{Path: "second.go"}}},
			})
		default:
			t.Fatalf("unexpected request path: %s", r.URL.String())
		}
	})

	stats, err := client.GetDiffStat("ws", "repo", 7)
	if err != nil {
		t.Fatalf("GetDiffStat() error = %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("expected 2 diffstat entries, got %d", len(stats))
	}
	if stats[0].New.Path != "first.go" || stats[1].New.Path != "second.go" {
		t.Fatalf("unexpected diffstat paths: %+v", stats)
	}
}
