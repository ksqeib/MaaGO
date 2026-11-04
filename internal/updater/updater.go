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

	"github.com/bodgit/sevenzip"

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
	coreAssetPattern = regexp.MustCompile(`^MaaCore-sdk-windows-x64-.*\.(zip|7z)$`)
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
	Target       string `json:"target"`
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
	mu          sync.RWMutex
	status      Status
	target      string
	coreDir     string
	updates     string
	emit        func(name string, payload any)
	busy        bool
	skipRuntime bool
}

// New creates an updater service for coreDir.
func New(coreDir string, emit func(name string, payload any)) *Service {
	updates := filepath.Join(filepath.Dir(coreDir), ".maago-updates")
	status := Status{Stage: StageIdle, Message: "idle"}
	if pending, ok, err := readPending(updates); err == nil && ok {
		target := pending.Kind
		if target == "resource" {
			target = "resource-overlay"
		}
		status = Status{
			Stage:        StageReady,
			Target:       target,
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
	if status.Target == "" {
		status.Target = s.target
	}
	s.status = status
	s.mu.Unlock()
	if s.emit != nil {
		s.emit("maago:update", status)
	}
}

func (s *Service) setTarget(target string) {
	s.mu.Lock()
	s.target = target
	s.status.Target = target
	s.mu.Unlock()
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
	s.setTarget("core")
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
	if err := s.begin("core"); err != nil {
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
		release, err = fetchCoreRelease(ctx, client, version)
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
	zipPath := filepath.Join(stageRoot, "core"+filepath.Ext(release.AssetName))

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
	if deps, ok, err := readCoreDependencies(stagingCore); err != nil {
		s.fail(err)
		return err
	} else if ok {
		if err := s.ensureRuntime(ctx, client, stageRoot, stagingCore, deps, cfg.Update.Mirror); err != nil {
			s.fail(err)
			return err
		}
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
	if err := s.begin("resource"); err != nil {
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

// ResourceVersions reports the installed base/overlay markers and the resource
// data date. It is used by the separate update sections.
type ResourceVersions struct {
	Base    string `json:"base"`
	Overlay string `json:"overlay"`
	Updated string `json:"updated"`
}

func (s *Service) Versions() ResourceVersions {
	resourceDir := filepath.Join(filepath.Dir(s.coreDir), "resource")
	read := func(name string) string {
		data, err := os.ReadFile(filepath.Join(resourceDir, name))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
	var meta struct {
		LastUpdated string `json:"last_updated"`
	}
	if data, err := os.ReadFile(filepath.Join(resourceDir, "version.json")); err == nil {
		_ = json.Unmarshal(data, &meta)
	}
	return ResourceVersions{
		Base:    read(".maago-base-version"),
		Overlay: read(".maago-overlay-version"),
		Updated: meta.LastUpdated,
	}
}

// CheckBase fetches the latest official MAA release, whose resource zip is the
// base resource (including platform_diff/PC).
func (s *Service) CheckBase(ctx context.Context) (ReleaseInfo, error) {
	s.setTarget("resource-base")
	cfg, err := config.Load()
	if err != nil {
		return ReleaseInfo{}, err
	}
	s.setStatus(Status{Stage: StageChecking, Message: "checking official base resource"})
	client, err := httpClient(cfg, "")
	if err != nil {
		s.fail(err)
		return ReleaseInfo{}, err
	}
	release, err := fetchOfficialLatest(ctx, client)
	if err != nil {
		s.fail(err)
		return ReleaseInfo{}, err
	}
	s.setStatus(Status{Stage: StageIdle, Message: "latest base " + release.Tag, Version: release.Tag})
	return release, nil
}

// CheckOverlay fetches the latest commit of MaaResource, which is applied as an
// overlay on top of the base resource.
func (s *Service) CheckOverlay(ctx context.Context) (ReleaseInfo, error) {
	s.setTarget("resource-overlay")
	cfg, err := config.Load()
	if err != nil {
		return ReleaseInfo{}, err
	}
	s.setStatus(Status{Stage: StageChecking, Message: "checking MaaResource overlay"})
	client, err := httpClient(cfg, "")
	if err != nil {
		s.fail(err)
		return ReleaseInfo{}, err
	}
	info, err := fetchOverlayCommit(ctx, client)
	if err != nil {
		s.fail(err)
		return ReleaseInfo{}, err
	}
	s.setStatus(Status{Stage: StageIdle, Message: "latest overlay " + info.Tag, Version: info.Tag})
	return info, nil
}

// DownloadBase stages the official MAA release resource as the base resource.
func (s *Service) DownloadBase(ctx context.Context) error {
	if err := s.begin("resource-base"); err != nil {
		return err
	}
	defer s.end()

	cfg, err := config.Load()
	if err != nil {
		s.fail(err)
		return err
	}
	client, err := httpClient(cfg, "")
	if err != nil {
		s.fail(err)
		return err
	}
	release, err := fetchOfficialLatest(ctx, client)
	if err != nil {
		s.fail(err)
		return err
	}
	stageRoot := filepath.Join(s.updates, "base-resource-"+time.Now().Format("20060102-150405"))
	cleanStaging(s.updates)
	_ = os.RemoveAll(stageRoot)
	if err := os.MkdirAll(stageRoot, 0o755); err != nil {
		s.fail(err)
		return err
	}
	zipPath := filepath.Join(stageRoot, "resource.zip")
	s.setStatus(Status{Stage: StageDownload, Target: "resource-base", Message: "downloading base resource " + release.Tag, Version: release.Tag, Total: release.Size})
	if err := s.download(ctx, client, withMirror(release.AssetURL, cfg.Update.Mirror), zipPath, release.Tag); err != nil {
		s.fail(err)
		return err
	}
	s.setStatus(Status{Stage: StageExtracting, Target: "resource-base", Message: "extracting base resource", Progress: 0})
	stagingResource := filepath.Join(stageRoot, "resource")
	if err := extractResource(zipPath, stagingResource, s.setStatus); err != nil {
		s.fail(err)
		return err
	}
	if _, err := os.Stat(filepath.Join(stagingResource, "version.json")); err != nil {
		err = errors.New("downloaded base resource has no version.json")
		s.fail(err)
		return err
	}
	if err := writePending(s.updates, pendingUpdate{Kind: "resource-base", Staging: stagingResource, Version: release.Tag}); err != nil {
		s.fail(err)
		return err
	}
	s.setStatus(Status{Stage: StageReady, Target: "resource-base", Message: "base resource " + release.Tag + " is ready; restart to apply", Version: release.Tag, Progress: 100, NeedsRestart: true})
	return nil
}

// DownloadOverlay stages the MaaResource main branch as an overlay.
func (s *Service) DownloadOverlay(ctx context.Context) error {
	if err := s.begin("resource-overlay"); err != nil {
		return err
	}
	defer s.end()

	cfg, err := config.Load()
	if err != nil {
		s.fail(err)
		return err
	}
	client, err := httpClient(cfg, "")
	if err != nil {
		s.fail(err)
		return err
	}
	info, err := fetchOverlayCommit(ctx, client)
	version := "main"
	if err == nil {
		version = info.Tag
	}
	stageRoot := filepath.Join(s.updates, "overlay-resource-"+time.Now().Format("20060102-150405"))
	cleanStaging(s.updates)
	_ = os.RemoveAll(stageRoot)
	if err := os.MkdirAll(stageRoot, 0o755); err != nil {
		s.fail(err)
		return err
	}
	zipPath := filepath.Join(stageRoot, "resource.zip")
	s.setStatus(Status{Stage: StageDownload, Target: "resource-overlay", Message: "downloading MaaResource overlay " + version, Version: version})
	if err := s.download(ctx, client, withMirror(resourceURL, cfg.Update.Mirror), zipPath, version); err != nil {
		s.fail(err)
		return err
	}
	s.setStatus(Status{Stage: StageExtracting, Target: "resource-overlay", Message: "extracting resource overlay", Progress: 0})
	stagingResource := filepath.Join(stageRoot, "resource")
	if err := extractResource(zipPath, stagingResource, s.setStatus); err != nil {
		s.fail(err)
		return err
	}
	if _, err := os.Stat(filepath.Join(stagingResource, "version.json")); err != nil {
		err = errors.New("downloaded resource overlay has no version.json")
		s.fail(err)
		return err
	}
	if err := writePending(s.updates, pendingUpdate{Kind: "resource-overlay", Staging: stagingResource, Version: version}); err != nil {
		s.fail(err)
		return err
	}
	s.setStatus(Status{Stage: StageReady, Target: "resource-overlay", Message: "resource overlay " + version + " is ready; restart to apply", Version: version, Progress: 100, NeedsRestart: true})
	return nil
}

func fetchOverlayCommit(ctx context.Context, client *http.Client) (ReleaseInfo, error) {
	const api = "https://api.github.com/repos/MaaAssistantArknights/MaaResource/commits/main"
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
		return ReleaseInfo{}, fmt.Errorf("check overlay failed: HTTP %d", response.StatusCode)
	}
	var payload struct {
		SHA    string `json:"sha"`
		Commit struct {
			Committer struct {
				Date string `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return ReleaseInfo{}, err
	}
	tag := payload.SHA
	if len(tag) > 7 {
		tag = tag[:7]
	}
	if tag == "" {
		tag = "main"
	}
	return ReleaseInfo{
		Tag:       tag,
		Version:   tag,
		AssetName: "MaaResource-main.zip",
		AssetURL:  resourceURL,
		Published: payload.Commit.Committer.Date,
	}, nil
}

func (s *Service) begin(target string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy {
		return errors.New("another update is already running")
	}
	if _, ok, err := readPending(s.updates); err == nil && ok {
		return errors.New("an update is already staged; restart MaaGO to apply it first")
	}
	s.busy = true
	s.target = target
	s.status.Target = target
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

func fetchCoreRelease(ctx context.Context, client *http.Client, version string) (ReleaseInfo, error) {
	tag := "v" + strings.TrimPrefix(strings.TrimSpace(version), "v")
	api := "https://api.github.com/repos/" + coreRepo + "/releases/tags/" + tag
	return fetchRelease(ctx, client, api, coreAssetPattern)
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
		if strings.Contains(strings.ToLower(asset.Name), "symbols") {
			continue
		}
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
	if strings.HasSuffix(strings.ToLower(zipPath), ".7z") {
		return extractCore7z(zipPath, staging, version, status)
	}
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
		metadata := filepath.Base(name) == "deps.json"
		if metadata {
			relative = "deps.json"
		} else if root != "" && relative == name {
			continue
		}
		// The SDK puts runtime DLLs in bin/ next to MaaCore.dll. Keep that flat
		// directory only; headers, PDBs and docs are not needed at runtime.
		keep := strings.HasSuffix(strings.ToLower(relative), ".dll") || metadata
		if strings.Contains(relative, "/") || !keep {
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

func extractCore7z(archivePath, staging, version string, status func(Status)) error {
	reader, err := sevenzip.OpenReader(archivePath)
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
		return errors.New("MaaCore.dll was not found in the 7z archive")
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
		metadata := filepath.Base(name) == "deps.json"
		if metadata {
			relative = "deps.json"
		} else if root != "" && relative == name {
			continue
		}
		keep := strings.HasSuffix(strings.ToLower(relative), ".dll") || metadata
		if strings.Contains(relative, "/") || !keep {
			continue
		}
		source, err := entry.Open()
		if err != nil {
			return err
		}
		destination := filepath.Join(staging, relative)
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			source.Close()
			return err
		}
		file, err := os.Create(destination)
		if err != nil {
			source.Close()
			return err
		}
		if _, err := io.Copy(file, source); err != nil {
			file.Close()
			source.Close()
			return err
		}
		if err := file.Close(); err != nil {
			source.Close()
			return err
		}
		source.Close()
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
		return errors.New("no DLL files were found in the 7z archive")
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
	// The SDK package only ships MaaCore.dll/MaaUtils.dll. Preserve the
	// third-party runtime DLLs, control units and adb from the old core so the
	// update does not leave a core that cannot load its dependencies.
	entries, err := os.ReadDir(sourceCore)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "resource" || name == "cache" {
			continue
		}
		target := filepath.Join(stagingCore, name)
		if _, err := os.Stat(target); err == nil {
			continue
		}
		if err := copyPath(filepath.Join(sourceCore, name), target); err != nil {
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
