package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestTokenProviderCachesAndRefreshes(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("username") != "20000000001MODDATOS" {
			t.Fatalf("unexpected username: %s", r.Form.Get("username"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "tok_test_123",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	}))
	defer srv.Close()

	tp := NewTokenProvider(Credentials{
		RUC:          "20000000001",
		SolUser:      "MODDATOS",
		SolPass:      "moddatos",
		ClientID:     "cid",
		ClientSecret: "csec",
	}, WithBaseURL(srv.URL))

	tok1, err := tp.Token(context.Background(), ScopeSIRE)
	if err != nil || tok1 != "tok_test_123" {
		t.Fatalf("unexpected token result: %q, err=%v", tok1, err)
	}
	tok2, err := tp.Token(context.Background(), ScopeSIRE)
	if err != nil || tok2 != "tok_test_123" {
		t.Fatalf("unexpected cached token result: %q, err=%v", tok2, err)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected 1 HTTP call due to caching, got %d", hits.Load())
	}
}
