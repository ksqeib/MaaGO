// Package updater downloads MaaCore releases and MaaResource updates.
//
// Core DLLs are locked while loaded, so a core update is staged next to the
// current core directory and applied by the next process start. Resource data
// is staged the same way to avoid changing files under a running core.
package updater

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"MaaGO/internal/config"
)

const (
	coreAPI       = "https://api.github.com/repos/ksqeib/MaaCore/releases/latest"
	coreRepo      = "ksqeib/MaaCore"
	maaReleaseAPI = "https://api.github.com/repos/MaaAssistantArknights/MaaAssistantArknights/releases/latest"
	resourceURL   = "https://codeload.github.com/MaaAssistantArknights/MaaResource/zip/refs/heads/main"
	userAgent     = "MaaGO-Updater"
)

var (
	coreAssetPattern = regexp.MustCompile(`^MaaCore-sdk-windows-x64-.*\.zip$`)
	maaAssetPattern  = regexp.MustCompile(`^MAA-v.+-win-x64\.zip$`)
)

// ReleaseInfo describes one MaaCore release asset.
type ReleaseInfo struct {
	Tag       string `json:"tag"`
	Version   string `json:"version"`
	AssetName string `json:"assetName"`
	AssetURL  string `json:"assetUrl"`
	Size      int64  `json:"size"`
	Published string `json:"published"`
}

// ProxyTestResult is the result of a proxy connectivity test.
type ProxyTestResult struct {
	OK        bool   `json:"ok"`
	Status    int    `json:"status"`
	LatencyMs int64  `json:"latencyMs"`
	Message   string `json:"message"`
}

// Status is the update progress snapshot emitted to the frontend.
type Status struct {
	Stage        string `json:"stage"`
	Message      string `json:"message"`
	Progress     int    `json:"progress"`
	Downloaded   int64  `json:"downloaded"`
	Total        int64  `json:"total"`
	Version      string `json:"version"`
	NeedsRestart bool   `json:"needsRestart"`
	Error        string `json:"error"`
}

const (
	StageIdle       = "idle"
	StageChecking   = "checking"
	StageDownload   = "downloading"
	StageExtracting = "extracting"
	StageReady      = "ready"
	StageError      = "error"
)

// Service owns update checks and downloads.
type Service struct {
	mu      sync.RWMutex
	status  Status
	coreDir string
	updates string
	emit    func(name string, payload any)
	busy    bool
}

// New creates an updater service for coreDir.
func New(coreDir string, emit func(name string, payload any)) *Service {
	updates := filepath.Join(filepath.Dir(coreDir), ".maago-updates")
	status := Status{Stage: StageIdle, Message: "idle"}
	if pending, ok, err := readPending(updates); err == nil && ok {
		status = Status{
			Stage:        StageReady,
			Message:      "pending " + pending.Kind + " update",
			Version:      pending.Version,
			NeedsRestart: true,
			Progress:     100,
		}
	}
	return &Service{
		status:  status,
		coreDir: coreDir,
		updates: updates,
		emit:    emit,
	}
}

// Status returns the current update state.
func (s *Service) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// Dir returns the staging directory.
func (s *Service) Dir() string {
	if s == nil {
		return ""
	}
	return s.updates
}

// Pending reports whether a staged update is waiting for a restart.
func (s *Service) Pending() bool {
	if s == nil {
		return false
	}
	_, ok, err := readPending(s.updates)
	return err == nil && ok
}

// Busy reports whether a download is running.
func (s *Service) Busy() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.busy
}

func (s *Service) setStatus(status Status) {
	s.mu.Lock()
	s.status = status
	s.mu.Unlock()
	if s.emit != nil {
		s.emit("maago:update", status)
	}
}

func (s *Service) fail(err error) {
	status := s.Status()
	status.Stage = StageError
	status.Error = err.Error()
	status.Message = err.Error()
	s.setStatus(status)
}

// Check fetches the latest release metadata.
func (s *Service) Check(ctx context.Context) (ReleaseInfo, error) {
	cfg, err := config.Load()
	if err != nil {
		return ReleaseInfo{}, err
	}
	s.setStatus(Status{Stage: StageChecking, Message: "checking latest release"})
	client, err := httpClient(cfg, "")
	if err != nil {
		s.fail(err)
		return ReleaseInfo{}, err
	}
	release, err := fetchLatest(ctx, client)
	if err != nil {
		s.fail(err)
		return ReleaseInfo{}, err
	}
	s.setStatus(Status{Stage: StageIdle, Message: "latest release " + release.Tag, Version: release.Tag})
	return release, nil
}

// DownloadCore downloads a release and stages it for the next start.
func (s *Service) DownloadCore(ctx context.Context, version string) error {
	if err := s.begin(); err != nil {
		return err
	}
	defer s.end()

	cfg, err := config.Load()
	if err != nil {
		s.fail(err)
		return err
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = strings.TrimSpace(cfg.Update.TargetVersion)
	}

	client, err := httpClient(cfg, "")
	if err != nil {
		s.fail(err)
		return err
	}

	var release ReleaseInfo
	if version == "" || strings.EqualFold(version, "latest") {
		s.setStatus(Status{Stage: StageChecking, Message: "resolving latest release"})
		release, err = fetchLatest(ctx, client)
	} else {
		release = releaseForVersion(version)
	}
	if err != nil {
		s.fail(err)
		return err
	}
	release.AssetURL = withMirror(release.AssetURL, cfg.Update.Mirror)

	stageRoot := filepath.Join(s.updates, "core-"+sanitize(release.Tag))
	cleanStaging(s.updates)
	_ = os.RemoveAll(stageRoot)
	if err := os.MkdirAll(stageRoot, 0o755); err != nil {
		s.fail(err)
		return err
	}
	zipPath := filepath.Join(stageRoot, "core.zip")

	s.setStatus(Status{
		Stage:    StageDownload,
		Message:  "downloading " + release.Tag,
		Version:  release.Tag,
		Total:    release.Size,
		Progress: 0,
	})
	if err := s.download(ctx, client, release.AssetURL, zipPath, release.Tag); err != nil {
		s.fail(err)
		return err
	}

	s.setStatus(Status{Stage: StageExtracting, Message: "extracting core", Version: release.Tag, Progress: 0})
	stagingCore := filepath.Join(stageRoot, "core")
	if err := extractCore(zipPath, stagingCore, release.Tag, s.setStatus); err != nil {
		s.fail(err)
		return err
	}
	if err := copyOptionalCoreFiles(s.coreDir, stagingCore); err != nil {
		s.fail(err)
		return err
	}
	if err := validateCore(stagingCore); err != nil {
		s.fail(err)
		return err
	}
	if err := writePending(s.updates, pendingUpdate{
		Kind:    "core",
		Staging: stagingCore,
		Version: release.Tag,
	}); err != nil {
		s.fail(err)
		return err
	}

	s.setStatus(Status{
		Stage:        StageReady,
		Message:      "core " + release.Tag + " is ready; restart to install",
		Version:      release.Tag,
		Progress:     100,
		NeedsRestart: true,
	})
	return nil
}

// DownloadResource downloads a full base resource when none is installed, or
// MaaResource hot-update files when the base resource already exists. The
// staged resource is applied next to the core directory on restart.
func (s *Service) DownloadResource(ctx context.Context) error {
	if err := s.begin(); err != nil {
		return err
	}
	defer s.end()

	cfg, err := config.Load()
	if err != nil {
		s.fail(err)
		return err
	}
	resourceDir := filepath.Join(filepath.Dir(s.coreDir), "resource")
	needBase := false
	if _, err := os.Stat(filepath.Join(resourceDir, "version.json")); err != nil {
		needBase = true
	}

	client, err := httpClient(cfg, "")
	if err != nil {
		s.fail(err)
		return err
	}

	stageRoot := filepath.Join(s.updates, "resource-"+time.Now().Format("20060102-150405"))
	cleanStaging(s.updates)
	_ = os.RemoveAll(stageRoot)
	if err := os.MkdirAll(stageRoot, 0o755); err != nil {
		s.fail(err)
		return err
	}
	zipPath := filepath.Join(stageRoot, "resource.zip")

	url := ""
	version := ""
	if needBase {
		release, err := fetchOfficialLatest(ctx, client)
		if err != nil {
			s.fail(err)
			return err
		}
		url = withMirror(release.AssetURL, cfg.Update.Mirror)
		version = release.Tag
		s.setStatus(Status{Stage: StageDownload, Message: "downloading base resource " + release.Tag, Version: release.Tag, Total: release.Size})
	} else {
		url = withMirror(resourceURL, cfg.Update.Mirror)
		s.setStatus(Status{Stage: StageDownload, Message: "downloading MaaResource"})
	}
	if err := s.download(ctx, client, url, zipPath, version); err != nil {
		s.fail(err)
		return err
	}

	s.setStatus(Status{Stage: StageExtracting, Message: "extracting resource", Progress: 0})
	stagingResource := filepath.Join(stageRoot, "resource")
	if err := extractResource(zipPath, stagingResource, s.setStatus); err != nil {
		s.fail(err)
		return err
	}
	if _, err := os.Stat(filepath.Join(stagingResource, "version.json")); err != nil {
		err = errors.New("downloaded resource has no version.json")
		s.fail(err)
		return err
	}
	if err := writePending(s.updates, pendingUpdate{
		Kind:    "resource",
		Staging: stagingResource,
		Version: version,
	}); err != nil {
		s.fail(err)
		return err
	}

	message := "resource update is ready; restart to apply"
	if needBase {
		message = "base resource is ready; restart to apply"
	}
	s.setStatus(Status{
		Stage:        StageReady,
		Message:      message,
		Progress:     100,
		NeedsRestart: true,
	})
	return nil
}

func (s *Service) begin() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("another update is already running")
	}
	if _, ok, err := readPending(s.updates); err == nil && ok {
		return errors.New("an update is already staged; restart MaaGO to apply it first")
	}
	s.busy = true
	return nil
}

func (s *Service) end() {
	s.mu.Lock()
	s.busy = false
	s.mu.Unlock()
}

func (s *Service) download(ctx context.Context, client *http.Client, url, destination, version string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", userAgent)
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", response.StatusCode)
	}
	total := response.ContentLength
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer file.Close()

	progress := &progressWriter{
		total:   total,
		version: version,
		status:  s.setStatus,
	}
	_, err = io.Copy(file, io.TeeReader(response.Body, progress))
	if err != nil {
		return err
	}
	status := s.Status()
	status.Progress = 100
	status.Downloaded = progress.written
	status.Total = total
	status.Stage = StageDownload
	status.Version = version
	status.Message = "download complete"
	s.setStatus(status)
	return nil
}

type progressWriter struct {
	written int64
	total   int64
	version string
	last    time.Time
	status  func(Status)
}

func (p *progressWriter) Write(data []byte) (int, error) {
	p.written += int64(len(data))
	now := time.Now()
	if now.Sub(p.last) < 150*time.Millisecond {
		return len(data), nil
	}
	p.last = now
	progress := 0
	if p.total > 0 {
		progress = int(p.written * 100 / p.total)
	}
	p.status(Status{
		Stage:      StageDownload,
		Message:    "downloading",
		Progress:   progress,
		Downloaded: p.written,
		Total:      p.total,
		Version:    p.version,
	})
	return len(data), nil
}

func fetchLatest(ctx context.Context, client *http.Client) (ReleaseInfo, error) {
	return fetchRelease(ctx, client, coreAPI, coreAssetPattern)
}

func fetchOfficialLatest(ctx context.Context, client *http.Client) (ReleaseInfo, error) {
	return fetchRelease(ctx, client, maaReleaseAPI, maaAssetPattern)
}

func fetchRelease(ctx context.Context, client *http.Client, api string, pattern *regexp.Regexp) (ReleaseInfo, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return ReleaseInfo{}, err
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := client.Do(request)
	if err != nil {
		return ReleaseInfo{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ReleaseInfo{}, fmt.Errorf("check update failed: HTTP %d", response.StatusCode)
	}
	var payload struct {
		TagName     string `json:"tag_name"`
		PublishedAt string `json:"published_at"`
		Assets      []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
			Size int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return ReleaseInfo{}, err
	}
	for _, asset := range payload.Assets {
		if pattern.MatchString(asset.Name) {
			return ReleaseInfo{
				Tag:       payload.TagName,
				Version:   payload.TagName,
				AssetName: asset.Name,
				AssetURL:  asset.URL,
				Size:      asset.Size,
				Published: payload.PublishedAt,
			}, nil
		}
	}
	return ReleaseInfo{}, errors.New("no matching release asset found")
}

func releaseForVersion(version string) ReleaseInfo {
	tag := "v" + strings.TrimPrefix(strings.TrimSpace(version), "v")
	asset := "MaaCore-sdk-windows-x64-" + tag + ".zip"
	return ReleaseInfo{
		Tag:       tag,
		Version:   tag,
		AssetName: asset,
		AssetURL:  "https://github.com/" + coreRepo + "/releases/download/" + tag + "/" + asset,
	}
}

func withMirror(rawURL, mirror string) string {
	mirror = strings.TrimSpace(mirror)
	if mirror == "" {
		return rawURL
	}
	return strings.TrimRight(mirror, "/") + "/" + rawURL
}

func httpClient(cfg config.AppConfig, overrideProxy string) (*http.Client, error) {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   20 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
	rawProxy := strings.TrimSpace(overrideProxy)
	if rawProxy == "" && cfg.Proxy.Enabled {
		rawProxy = strings.TrimSpace(cfg.Proxy.URL)
	}
	if rawProxy != "" {
		if !strings.Contains(rawProxy, "://") {
			rawProxy = "http://" + rawProxy
		}
		proxyURL, err := url.Parse(rawProxy)
		if err != nil || proxyURL.Host == "" {
			return nil, fmt.Errorf("invalid proxy URL: %s", rawProxy)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{Transport: transport}, nil
}

// NewHTTPClient builds the same proxy-aware client used by the updater.
func NewHTTPClient(cfg config.AppConfig) (*http.Client, error) {
	return httpClient(cfg, "")
}

// TestProxy checks GitHub API reachability through a proxy URL.
func TestProxy(ctx context.Context, proxyURL string) ProxyTestResult {
	cfg, _ := config.Load()
	client, err := httpClient(cfg, proxyURL)
	if err != nil {
		return ProxyTestResult{OK: false, Message: err.Error()}
	}
	start := time.Now()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com", nil)
	if err != nil {
		return ProxyTestResult{OK: false, Message: err.Error()}
	}
	request.Header.Set("User-Agent", userAgent)
	response, err := client.Do(request)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return ProxyTestResult{OK: false, LatencyMs: latency, Message: err.Error()}
	}
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 500 {
		return ProxyTestResult{
			OK:        true,
			Status:    response.StatusCode,
			LatencyMs: latency,
			Message:   fmt.Sprintf("HTTP %d · %d ms", response.StatusCode, latency),
		}
	}
	return ProxyTestResult{
		OK:        false,
		Status:    response.StatusCode,
		LatencyMs: latency,
		Message:   fmt.Sprintf("HTTP %d", response.StatusCode),
	}
}

func extractCore(zipPath, staging, version string, status func(Status)) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	root := ""
	found := false
	for _, entry := range reader.File {
		name := strings.ReplaceAll(entry.Name, "\\", "/")
		if name == "MaaCore.dll" || strings.HasSuffix(name, "/MaaCore.dll") {
			root = strings.TrimSuffix(name, "MaaCore.dll")
			found = true
			break
		}
	}
	if !found {
		return errors.New("MaaCore.dll was not found in the archive")
	}

	_ = os.RemoveAll(staging)
	total := len(reader.File)
	copied := 0
	for index, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		name := strings.ReplaceAll(entry.Name, "\\", "/")
		relative := strings.TrimPrefix(name, root)
		if root != "" && relative == name {
			continue
		}
		// The SDK puts runtime DLLs in bin/ next to MaaCore.dll. Keep that flat
		// directory only; headers, PDBs and docs are not needed at runtime.
		if strings.Contains(relative, "/") || !strings.HasSuffix(strings.ToLower(relative), ".dll") {
			continue
		}
		if err := extractEntry(entry, staging, relative); err != nil {
			return err
		}
		copied++
		if index%10 == 0 {
			status(Status{
				Stage:    StageExtracting,
				Message:  "extracting core",
				Progress: index * 100 / max(total, 1),
				Version:  version,
			})
		}
	}
	if copied == 0 {
		return errors.New("no runtime DLLs were found next to MaaCore.dll")
	}
	return nil
}

func extractResource(zipPath, staging string, status func(Status)) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	root := ""
	found := false
	for _, entry := range reader.File {
		if strings.HasSuffix(entry.Name, "resource/version.json") {
			root = strings.TrimSuffix(entry.Name, "resource/version.json")
			found = true
			break
		}
	}
	if !found {
		return errors.New("resource/version.json was not found in the archive")
	}
	_ = os.RemoveAll(staging)
	total := len(reader.File)
	for index, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		relative := strings.TrimPrefix(entry.Name, root)
		if root != "" && relative == entry.Name {
			continue
		}
		if !strings.HasPrefix(relative, "resource/") {
			continue
		}
		relative = strings.TrimPrefix(relative, "resource/")
		if err := extractEntry(entry, staging, relative); err != nil {
			return err
		}
		if index%100 == 0 {
			status(Status{
				Stage:    StageExtracting,
				Message:  "extracting resource",
				Progress: index * 100 / max(total, 1),
			})
		}
	}
	return nil
}

func extractEntry(entry *zip.File, destination, relative string) error {
	target := filepath.Join(destination, filepath.FromSlash(relative))
	cleanDestination := filepath.Clean(destination) + string(os.PathSeparator)
	if !strings.HasPrefix(filepath.Clean(target), cleanDestination) {
		return fmt.Errorf("unsafe archive path: %s", entry.Name)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	source, err := entry.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, source)
	return err
}

func copyOptionalCoreFiles(sourceCore, stagingCore string) error {
	// The SDK package does not ship the Win32 control unit; preserve it from
	// the old core so PC-window mode keeps working across core updates.
	for _, name := range []string{"adb", "README.md", "MaaWin32ControlUnit.dll", "MaaAdbControlUnit.dll"} {
		source := filepath.Join(sourceCore, name)
		if _, err := os.Stat(source); err != nil {
			continue
		}
		target := filepath.Join(stagingCore, name)
		if err := copyPath(source, target); err != nil {
			return err
		}
	}
	return nil
}

func validateCore(coreDir string) error {
	for _, name := range []string{"MaaCore.dll", "MaaUtils.dll"} {
		if _, err := os.Stat(filepath.Join(coreDir, name)); err != nil {
			return fmt.Errorf("staged core has no %s", name)
		}
	}
	return nil
}

func copyPath(source, target string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(source, target)
	}
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if info.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		return copyFile(path, destination)
	})
}

func copyFile(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	output, err := os.Create(target)
	if err != nil {
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, input)
	return err
}

func sanitize(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func cleanStaging(updatesDir string) {
	entries, err := os.ReadDir(updatesDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			_ = os.RemoveAll(filepath.Join(updatesDir, entry.Name()))
		}
	}
}
