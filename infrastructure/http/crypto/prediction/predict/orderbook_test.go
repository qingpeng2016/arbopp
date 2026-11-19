package predict

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetOrderbooks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/markets/orderbooks" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		ids := r.URL.Query()["ids"]
		if len(ids) != 2 || ids[0] != "4211675" || ids[1] != "4211677" {
			t.Fatalf("ids=%v", ids)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":[{"marketId":4211675,"asks":[[0.97,200]],"bids":[[0.01,600]]},{"marketId":4211677,"asks":[[0.5,10]],"bids":[[0.49,8]]}]}`))
	}))
	defer srv.Close()

	c := NewClientWithOptions(srv.URL, "k", &http.Client{Timeout: 2 * time.Second})
	got, err := c.GetOrderbooks(context.Background(), []string{"4211675", "4211677", "4211675"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].MarketID != 4211675 || got[1].MarketID != 4211677 {
		t.Fatalf("got=%#v", got)
	}
}

func TestGetOrderbooksRejects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"message":"too many ids"}`))
	}))
	defer srv.Close()
	c := NewClientWithOptions(srv.URL, "k", &http.Client{Timeout: 2 * time.Second})
	_, err := c.GetOrderbooks(context.Background(), []string{"1"})
	if err == nil || !strings.Contains(err.Error(), "too many ids") {
		t.Fatalf("err=%v", err)
	}
}
