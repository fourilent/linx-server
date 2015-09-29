package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/zenazn/goji"
)

func TestSetup(t *testing.T) {
	Config.siteURL = "http://linx.example.org/"
	Config.filesDir = path.Join(os.TempDir(), randomString(10))
	Config.metaDir = Config.filesDir + "_meta"
	Config.noLogs = true
	Config.siteName = "linx"
	setup()
}

func TestIndex(t *testing.T) {
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	goji.DefaultMux.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "file-uploader") {
		t.Fatal("String 'file-uploader' not found in index response")
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/url/should/not/exist", nil)
	if err != nil {
		t.Fatal(err)
	}

	goji.DefaultMux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("Expected 404, got %d", w.Code)
	}
}

func TestFileNotFound(t *testing.T) {
	w := httptest.NewRecorder()

	filename := randomString(10)

	req, err := http.NewRequest("GET", "/selif/"+filename, nil)
	if err != nil {
		t.Fatal(err)
	}

	goji.DefaultMux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("Expected 404, got %d", w.Code)
	}
}

func TestDisplayNotFound(t *testing.T) {
	w := httptest.NewRecorder()

	filename := randomString(10)

	req, err := http.NewRequest("GET", "/"+filename, nil)
	if err != nil {
		t.Fatal(err)
	}

	goji.DefaultMux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("Expected 404, got %d", w.Code)
	}
}

func TestPostBodyUpload(t *testing.T) {
	w := httptest.NewRecorder()

	filename := randomString(10) + ".ext"

	req, err := http.NewRequest("POST", "/upload", strings.NewReader("File content"))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	params := req.URL.Query()
	params.Add("qqfile", filename)
	req.URL.RawQuery = params.Encode()

	goji.DefaultMux.ServeHTTP(w, req)

	if w.Code != 301 {
		t.Fatalf("Status code is not 301, but %d", w.Code)
	}
	if w.Header().Get("Location") != "/"+filename {
		t.Fatalf("Was redirected to %s instead of /%s", w.Header().Get("Location"), filename)
	}
}

func TestPostEmptyBodyUpload(t *testing.T) {
	w := httptest.NewRecorder()

	filename := randomString(10) + ".ext"

	req, err := http.NewRequest("POST", "/upload", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	params := req.URL.Query()
	params.Add("qqfile", filename)
	req.URL.RawQuery = params.Encode()

	goji.DefaultMux.ServeHTTP(w, req)

	if w.Code == 301 {
		t.Fatal("Status code is 301")
	}

	if !strings.Contains(w.Body.String(), "Oops! Something went wrong.") {
		t.Fatal("Response doesn't contain'Oops! Something went wrong.'")
	}
}

func TestPostBodyRandomizeUpload(t *testing.T) {
	w := httptest.NewRecorder()

	filename := randomString(10) + ".ext"

	req, err := http.NewRequest("POST", "/upload", strings.NewReader("File content"))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	params := req.URL.Query()
	params.Add("qqfile", filename)
	params.Add("randomize", "true")
	req.URL.RawQuery = params.Encode()

	goji.DefaultMux.ServeHTTP(w, req)

	if w.Code != 301 {
		t.Fatalf("Status code is not 301, but %d", w.Code)
	}
	if w.Header().Get("Location") == "/"+filename {
		t.Fatalf("Was redirected to %s instead of something random", filename)
	}
}

func TestPostBodyExpireUpload(t *testing.T) {
	// Dependant on json info on display url to check expiry
}

func TestPutUpload(t *testing.T) {
	w := httptest.NewRecorder()

	filename := randomString(10) + ".ext"

	req, err := http.NewRequest("PUT", "/upload/"+filename, strings.NewReader("File content"))
	if err != nil {
		t.Fatal(err)
	}

	goji.DefaultMux.ServeHTTP(w, req)

	if w.Body.String() != Config.siteURL+filename {
		t.Fatal("Response was not expected URL")
	}
}

func TestPutRandomizedUpload(t *testing.T) {
	w := httptest.NewRecorder()

	filename := randomString(10) + ".ext"

	req, err := http.NewRequest("PUT", "/upload/"+filename, strings.NewReader("File content"))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("X-Randomized-Barename", "yes")

	goji.DefaultMux.ServeHTTP(w, req)

	if w.Body.String() == Config.siteURL+filename {
		t.Fatal("Filename was not random")
	}
}

func TestPutEmptyUpload(t *testing.T) {
	w := httptest.NewRecorder()

	filename := randomString(10) + ".ext"

	req, err := http.NewRequest("PUT", "/upload/"+filename, strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("X-Randomized-Barename", "yes")

	goji.DefaultMux.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "Oops! Something went wrong.") {
		t.Fatal("Response doesn't contain'Oops! Something went wrong.'")
	}
}

func TestPutJSONUpload(t *testing.T) {
	type RespJSON struct {
		Filename  string
		Url       string
		DeleteKey string
		Expiry    string
		Size      string
	}
	var myjson RespJSON

	w := httptest.NewRecorder()

	filename := randomString(10) + ".ext"

	req, err := http.NewRequest("PUT", "/upload/"+filename, strings.NewReader("File content"))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Accept", "application/json")

	goji.DefaultMux.ServeHTTP(w, req)

	err = json.Unmarshal([]byte(w.Body.String()), &myjson)
	if err != nil {
		t.Fatal(err)
	}

	if myjson.Filename != filename {
		t.Fatal("Filename was not provided one but " + myjson.Filename)
	}
}

func TestPutRandomizedJSONUpload(t *testing.T) {
	type RespJSON struct {
		Filename  string
		Url       string
		DeleteKey string
		Expiry    string
		Size      string
	}
	var myjson RespJSON

	w := httptest.NewRecorder()

	filename := randomString(10) + ".ext"

	req, err := http.NewRequest("PUT", "/upload/"+filename, strings.NewReader("File content"))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Randomized-Barename", "yes")

	goji.DefaultMux.ServeHTTP(w, req)

	err = json.Unmarshal([]byte(w.Body.String()), &myjson)
	if err != nil {
		t.Fatal(err)
	}

	if myjson.Filename == filename {
		t.Fatal("Filename was not random ")
	}
}

func TestPutExpireJSONUpload(t *testing.T) {
	type RespJSON struct {
		Filename  string
		Url       string
		DeleteKey string
		Expiry    string
		Size      string
	}
	var myjson RespJSON

	w := httptest.NewRecorder()

	filename := randomString(10) + ".ext"

	req, err := http.NewRequest("PUT", "/upload/"+filename, strings.NewReader("File content"))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-File-Expiry", "600")

	goji.DefaultMux.ServeHTTP(w, req)

	err = json.Unmarshal([]byte(w.Body.String()), &myjson)
	if err != nil {
		t.Fatal(err)
	}

	expiry, err := strconv.Atoi(myjson.Expiry)
	if err != nil {
		t.Fatal("Expiry was not an integer")
	}
	if expiry < 1 {
		t.Fatal("Expiry was not set")
	}
}

func TestShutdown(t *testing.T) {
	os.RemoveAll(Config.filesDir)
	os.RemoveAll(Config.metaDir)
}