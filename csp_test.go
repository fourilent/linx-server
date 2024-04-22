package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/zenazn/goji"
)

var testCSPHeaders = map[string]string{
	"Content-Security-Policy": defaultCSPOptions.policy,
	"Referrer-Policy":         defaultCSPOptions.referrerPolicy,
}

func TestContentSecurityPolicy(t *testing.T) {
	Config.siteURL = "http://linx.example.org/"
	Config.filesDir = path.Join(os.TempDir(), generateBarename())
	Config.metaDir = Config.filesDir + "_meta"
	Config.selifPath = "selif"
	mux := setup()

	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	goji.Use(ContentSecurityPolicy(defaultCSPOptions))

	mux.ServeHTTP(w, req)

	for k, v := range testCSPHeaders {
		if w.Header().Get(k) != v {
			// t.Fatalf("%s header did not match expected value set by middleware", k)
			t.Fatalf("Expected %s header to be %s, got %s", k, v, w.Header().Get(k))
		}
	}
}
