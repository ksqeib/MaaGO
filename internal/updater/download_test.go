package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadResumesPartialFile(t *testing.T) {
	var rangeHeader string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		rangeHeader = request.Header.Get("Range")
		writer.Header().Set("Content-Length", "5")
		writer.WriteHeader(http.StatusPartialContent)
		_, _ = writer.Write([]byte("world"))
	}))
	defer server.Close()

	dir := t.TempDir()
	destination := filepath.Join(dir, "asset.bin")
	if err := os.WriteFile(destination, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := New(filepath.Join(dir, "core"), nil)
	if err := service.downloadOnce(context.Background(), server.Client(), server.URL, destination, "v1", 10); err != nil {
		t.Fatal(err)
	}
	if rangeHeader != "bytes=5-" {
		t.Fatalf("Range = %q, want bytes=5-", rangeHeader)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "helloworld" {
		t.Fatalf("content = %q", data)
	}
}

func TestDownloadRestartsWhenServerIgnoresRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", "5")
		_, _ = writer.Write([]byte("fresh"))
	}))
	defer server.Close()

	dir := t.TempDir()
	destination := filepath.Join(dir, "asset.bin")
	if err := os.WriteFile(destination, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := New(filepath.Join(dir, "core"), nil)
	if err := service.downloadOnce(context.Background(), server.Client(), server.URL, destination, "v1", 5); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fresh" {
		t.Fatalf("content = %q, want fresh", data)
	}
}
