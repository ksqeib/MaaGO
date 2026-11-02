package stages

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// FetchActivity downloads MAA's StageActivityV2 snapshot and writes it
// atomically to cachePath. The payload is small (a few KB).
func FetchActivity(ctx context.Context, client *http.Client, cachePath string) error {
	if client == nil {
		return fmt.Errorf("nil http client")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ActivityURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "MaaGO-stage-catalog")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("stage activity api returned %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return err
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("invalid stage activity payload: %w", err)
	}
	if _, ok := probe["Official"]; !ok {
		return fmt.Errorf("stage activity payload has no Official section")
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return err
	}
	temp := cachePath + ".tmp"
	if err := os.WriteFile(temp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temp, cachePath)
}
