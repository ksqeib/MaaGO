package updater

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"MaaGO/internal/config"
)

// coreDependencies mirrors the subset of the MaaCore deps.json that describes
// runtime DLLs. Build-time dependencies (Boost, zlib) are ignored.
type coreDependencies struct {
	SchemaVersion int                        `json:"schemaVersion"`
	Dependencies  map[string]coreDependency  `json:"dependencies"`
	RuntimeFiles  map[string]string          `json:"runtimeFiles"`
	Runtime       map[string]runtimePlatform `json:"runtime"`
}

type coreDependency struct {
	Version string `json:"version"`
	Flavor  string `json:"flavor"`
	Linkage string `json:"linkage"`
}

// runtimePlatform is the schema v2 runtime section.
type runtimePlatform struct {
	Bundle runtimeBundle              `json:"bundle"`
	Files  map[string]runtimeFileSpec `json:"files"`
}

type runtimeBundle struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type runtimeFileSpec struct {
	Dependency string `json:"dependency"`
	Version    string `json:"version"`
	Flavor     string `json:"flavor"`
	SHA256     string `json:"sha256"`
}

type runtimePackFile struct {
	Dependency  string
	ArchivePath string
	Size        int64
	SHA256      string
	Version     string
	Flavor      string
}

type runtimePack struct {
	Name         string
	URL          string
	SHA256       string
	Size         int64
	Dependencies map[string]string
	Flavors      map[string]string
	Files        map[string]runtimePackFile
}

// runtimePacks lists known-good, self-contained runtime collections. The SDK
// release no longer ships OpenCV/ONNX Runtime/DirectML/FastDeploy, so MaaGO
// extracts them from a pinned full package instead of trusting whatever DLLs
// happen to be next to the old core.
//
// v0.1.3 is kept as a legacy fallback for old schema-v1 deps.json files. New
// schema-v2 cores provide runtime.windows-x64.bundle directly.
var runtimePacks = []runtimePack{
	{
		Name:   "ksqeib/MaaCore v0.1.3 full SDK runtime",
		URL:    "https://github.com/ksqeib/MaaCore/releases/download/v0.1.3/MaaCore-sdk-windows-x64-v0.1.3.zip",
		SHA256: "1bdcf752675c1381382a79f5e289fef6bd5f84a96a9a4eec25daafa325d68d88",
		Size:   71588734,
		Dependencies: map[string]string{
			"opencv":      "4.12.0",
			"onnxruntime": "1.19.2",
			"directml":    "1.15.4",
			"fastdeploy":  "1.0.7",
		},
		Flavors: map[string]string{
			"onnxruntime": "directml",
		},
		Files: map[string]runtimePackFile{
			"opencv_world4_maa.dll": {
				Dependency:  "opencv",
				ArchivePath: "bin/opencv_world4_maa.dll",
				Size:        64217088,
				SHA256:      "0455560c75bc456b5f758c94af653b6f0397ed8a28b9605cf52c62e7c8f18781",
			},
			"onnxruntime_maa.dll": {
				Dependency:  "onnxruntime",
				ArchivePath: "bin/onnxruntime_maa.dll",
				Size:        14321184,
				SHA256:      "410de3464172dae2945a4663f274e34c59965bc8233b0411de16baab56dda1a7",
			},
			"DirectML.dll": {
				Dependency:  "directml",
				ArchivePath: "bin/DirectML.dll",
				Size:        18527776,
				SHA256:      "9c9e6d822561c6c41b90e6994b3e8857cf1d66dbfb1e0c4c799c7c89b4e92da1",
			},
			"fastdeploy_ppocr_maa.dll": {
				Dependency:  "fastdeploy",
				ArchivePath: "bin/fastdeploy_ppocr_maa.dll",
				Size:        1169920,
				SHA256:      "52591c163ed7d6b8c0196a54ca5168fc1eb3b785f06253e200424771e9183fb6",
			},
		},
	},
}

// RuntimeFileStatus describes one runtime DLL required by deps.json.
type RuntimeFileStatus struct {
	Name       string `json:"name"`
	Dependency string `json:"dependency"`
	Required   string `json:"required"`
	Installed  bool   `json:"installed"`
	Match      bool   `json:"match"`
}

// RuntimeStatus reports whether the third-party DLLs in core/ satisfy the
// dependency versions recorded by the installed MaaCore.
type RuntimeStatus struct {
	Available  bool                `json:"available"`
	Verified   bool                `json:"verified"`
	OK         bool                `json:"ok"`
	CanRepair  bool                `json:"canRepair"`
	Message    string              `json:"message"`
	Files      []RuntimeFileStatus `json:"files"`
	Missing    []string            `json:"missing"`
	Mismatched []string            `json:"mismatched"`
}

func readCoreDependencies(coreDir string) (coreDependencies, bool, error) {
	data, err := os.ReadFile(filepath.Join(coreDir, "deps.json"))
	if os.IsNotExist(err) {
		return coreDependencies{}, false, nil
	}
	if err != nil {
		return coreDependencies{}, false, err
	}
	var deps coreDependencies
	if err := json.Unmarshal(data, &deps); err != nil {
		return coreDependencies{}, false, fmt.Errorf("parse core deps.json: %w", err)
	}
	return deps, true, nil
}

func (deps coreDependencies) runtimeFileMap() map[string]string {
	files := make(map[string]string, len(deps.RuntimeFiles))
	for name, key := range deps.RuntimeFiles {
		files[name] = key
	}
	if platform, ok := deps.Runtime["windows-x64"]; ok {
		for name, spec := range platform.Files {
			files[name] = spec.Dependency
		}
	}
	return files
}

func (deps coreDependencies) runtimeDependencyKeys() []string {
	seen := map[string]bool{}
	for _, key := range deps.runtimeFileMap() {
		if key != "" {
			seen[key] = true
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (deps coreDependencies) dependencyVersion(key string) string {
	if dependency, ok := deps.Dependencies[key]; ok && dependency.Version != "" {
		return dependency.Version
	}
	if platform, ok := deps.Runtime["windows-x64"]; ok {
		for _, spec := range platform.Files {
			if spec.Dependency == key && spec.Version != "" {
				return spec.Version
			}
		}
	}
	return ""
}

func (deps coreDependencies) versionLabel() string {
	keys := deps.runtimeDependencyKeys()
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := key
		if version := deps.dependencyVersion(key); version != "" {
			value += " " + version
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		return "runtime"
	}
	return strings.Join(parts, ", ")
}

func matchRuntimePack(deps coreDependencies) *runtimePack {
	if len(deps.RuntimeFiles) == 0 {
		return nil
	}
	for index := range runtimePacks {
		pack := &runtimePacks[index]
		matches := true
		for name, key := range deps.RuntimeFiles {
			file, ok := pack.Files[name]
			if !ok || file.Dependency != key {
				matches = false
				break
			}
			dependency, ok := deps.Dependencies[key]
			if !ok {
				matches = false
				break
			}
			if dependency.Version != "" && pack.Dependencies[key] != dependency.Version {
				matches = false
				break
			}
			if dependency.Flavor != "" && pack.Flavors[key] != dependency.Flavor {
				matches = false
				break
			}
		}
		if matches {
			return pack
		}
	}
	return nil
}

func runtimePackFor(deps coreDependencies) *runtimePack {
	if platform, ok := deps.Runtime["windows-x64"]; ok && platform.Bundle.URL != "" {
		pack := &runtimePack{
			Name:         "MaaCore runtime bundle",
			URL:          platform.Bundle.URL,
			SHA256:       platform.Bundle.SHA256,
			Size:         platform.Bundle.Size,
			Dependencies: map[string]string{},
			Flavors:      map[string]string{},
			Files:        map[string]runtimePackFile{},
		}
		for name, spec := range platform.Files {
			pack.Files[name] = runtimePackFile{
				Dependency:  spec.Dependency,
				ArchivePath: name,
				SHA256:      spec.SHA256,
				Version:     spec.Version,
				Flavor:      spec.Flavor,
			}
			if spec.Dependency != "" && spec.Version != "" {
				pack.Dependencies[spec.Dependency] = spec.Version
			}
			if spec.Dependency != "" && spec.Flavor != "" {
				pack.Flavors[spec.Dependency] = spec.Flavor
			}
		}
		return pack
	}
	return matchRuntimePack(deps)
}

func inspectRuntimeFiles(coreDir string, deps coreDependencies, pack *runtimePack, verifyHash bool) (missing, mismatched []string, files []RuntimeFileStatus) {
	allFiles := deps.runtimeFileMap()
	names := make([]string, 0, len(allFiles))
	for name := range allFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		key := allFiles[name]
		required := ""
		if dependency, ok := deps.Dependencies[key]; ok {
			required = dependency.Version
		}
		if pack != nil {
			if expected, ok := pack.Files[name]; ok && expected.Version != "" {
				required = expected.Version
			}
		}
		state := RuntimeFileStatus{
			Name:       name,
			Dependency: key,
			Required:   required,
		}
		info, err := os.Stat(filepath.Join(coreDir, name))
		if err != nil || info.IsDir() {
			missing = append(missing, name)
			files = append(files, state)
			continue
		}
		state.Installed = true
		state.Match = true
		if pack != nil {
			expected, ok := pack.Files[name]
			if !ok {
				state.Match = false
			} else if expected.Size > 0 && info.Size() != expected.Size {
				state.Match = false
			} else if verifyHash && expected.SHA256 != "" {
				sum, err := hashFile(filepath.Join(coreDir, name))
				if err != nil || !strings.EqualFold(sum, expected.SHA256) {
					state.Match = false
				}
			}
		}
		if !state.Match {
			mismatched = append(mismatched, name)
		}
		files = append(files, state)
	}
	return missing, mismatched, files
}

func (s *Service) RuntimeStatus() RuntimeStatus {
	if s == nil {
		return RuntimeStatus{}
	}
	deps, ok, err := readCoreDependencies(s.coreDir)
	if err != nil {
		return RuntimeStatus{Message: err.Error()}
	}
	if !ok {
		return RuntimeStatus{
			Available: false,
			OK:        true,
			Message:   "core has no deps.json; runtime check skipped",
		}
	}
	pack := runtimePackFor(deps)
	missing, mismatched, files := inspectRuntimeFiles(s.coreDir, deps, pack, false)
	status := RuntimeStatus{
		Available:  true,
		Verified:   pack != nil,
		OK:         len(missing) == 0 && len(mismatched) == 0,
		Files:      files,
		Missing:    missing,
		Mismatched: mismatched,
	}
	switch {
	case len(missing) > 0:
		status.Message = "missing runtime files: " + strings.Join(missing, ", ")
	case !status.Verified:
		status.Message = "runtime dependencies are present but cannot be verified: no matching runtime pack for this dependency set"
	case status.OK:
		status.Message = "runtime dependencies are complete"
	default:
		status.Message = "runtime files do not match the required versions: " + strings.Join(mismatched, ", ")
	}
	if !status.OK {
		status.CanRepair = pack != nil
		if pack == nil {
			status.Message += "; no runtime pack is available for this dependency set"
		}
	}
	return status
}

// ensureRuntime makes a staged core self-contained. It is called after the new
// core DLLs are extracted and old core files are copied over.
func (s *Service) ensureRuntime(ctx context.Context, client *http.Client, stageRoot, stagingCore string, deps coreDependencies, mirror string) error {
	if s.skipRuntime || len(deps.runtimeFileMap()) == 0 {
		return nil
	}
	pack := runtimePackFor(deps)
	missing, mismatched, _ := inspectRuntimeFiles(stagingCore, deps, pack, true)
	if len(missing) == 0 && len(mismatched) == 0 {
		writeRuntimeMarker(stagingCore, deps, pack, "preserved from previous core")
		return nil
	}
	if pack == nil {
		return fmt.Errorf("core requires %s, but no runtime pack is available for this dependency set; install the full MAA release or update MaaGO", deps.versionLabel())
	}
	s.setStatus(Status{
		Stage:   StageDownload,
		Message: "downloading runtime dependencies",
		Version: deps.versionLabel(),
		Total:   pack.Size,
	})
	if err := s.downloadRuntimePack(ctx, client, pack, deps, stageRoot, stagingCore, mirror); err != nil {
		return err
	}
	writeRuntimeMarker(stagingCore, deps, pack, pack.Name)
	return nil
}

func (s *Service) downloadRuntimePack(ctx context.Context, client *http.Client, pack *runtimePack, deps coreDependencies, stageRoot, destination, mirror string) error {
	archive := filepath.Join(stageRoot, "runtime-pack.zip")
	if err := s.download(ctx, client, withMirror(pack.URL, mirror), archive, deps.versionLabel(), pack.Size); err != nil {
		return err
	}
	sum, err := hashFile(archive)
	if err != nil {
		return err
	}
	if !strings.EqualFold(sum, pack.SHA256) {
		_ = os.Remove(archive)
		return errors.New("runtime pack checksum mismatch")
	}
	s.setStatus(Status{
		Stage:   StageExtracting,
		Message: "extracting runtime dependencies",
		Version: deps.versionLabel(),
	})
	if err := extractRuntimeArchive(archive, destination, deps, pack); err != nil {
		return err
	}
	_ = os.Remove(archive)
	return nil
}

func extractRuntimeArchive(archive, destination string, deps coreDependencies, pack *runtimePack) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()

	type wantedFile struct {
		Name string
	}
	wanted := map[string]wantedFile{}
	for name := range deps.runtimeFileMap() {
		file, ok := pack.Files[name]
		if !ok {
			return fmt.Errorf("runtime pack has no entry for %s", name)
		}
		wanted[normalizeArchivePath(file.ArchivePath)] = wantedFile{Name: name}
	}

	found := map[string]bool{}
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		item, ok := wanted[normalizeArchivePath(entry.Name)]
		if !ok {
			continue
		}
		if err := extractEntry(entry, destination, item.Name); err != nil {
			return err
		}
		found[item.Name] = true
	}
	for name := range deps.runtimeFileMap() {
		if !found[name] {
			return fmt.Errorf("runtime pack is missing %s", name)
		}
		expected := pack.Files[name]
		path := filepath.Join(destination, name)
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if expected.Size > 0 && info.Size() != expected.Size {
			return fmt.Errorf("runtime file %s has size %d, expected %d", name, info.Size(), expected.Size)
		}
		if expected.SHA256 != "" {
			sum, err := hashFile(path)
			if err != nil {
				return err
			}
			if !strings.EqualFold(sum, expected.SHA256) {
				return fmt.Errorf("runtime file %s checksum mismatch", name)
			}
		}
	}
	return nil
}

func normalizeArchivePath(name string) string {
	return strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "/")
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

type runtimeMarker struct {
	Source       string            `json:"source"`
	InstalledAt  string            `json:"installedAt"`
	Dependencies map[string]string `json:"dependencies"`
}

func writeRuntimeMarker(coreDir string, deps coreDependencies, pack *runtimePack, source string) {
	versions := map[string]string{}
	for _, key := range deps.runtimeDependencyKeys() {
		value := ""
		if dependency, ok := deps.Dependencies[key]; ok {
			value = dependency.Version
			if dependency.Flavor != "" {
				value += "/" + dependency.Flavor
			}
		}
		if value == "" && pack != nil {
			value = pack.Dependencies[key]
		}
		versions[key] = value
	}
	data, err := json.MarshalIndent(runtimeMarker{
		Source:       source,
		InstalledAt:  time.Now().Format(time.RFC3339),
		Dependencies: versions,
	}, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(coreDir, ".maago-runtime.json"), data, 0o644)
}

// DownloadRuntime stages a runtime pack for the currently installed core. It is
// used for repairing an existing installation without changing MaaCore itself.
func (s *Service) DownloadRuntime(ctx context.Context) error {
	if err := s.begin("runtime"); err != nil {
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
	deps, ok, err := readCoreDependencies(s.coreDir)
	if err != nil {
		s.fail(err)
		return err
	}
	if !ok {
		err = errors.New("core has no deps.json; cannot determine runtime dependencies")
		s.fail(err)
		return err
	}
	pack := runtimePackFor(deps)
	if pack == nil {
		err = fmt.Errorf("no runtime pack is available for %s", deps.versionLabel())
		s.fail(err)
		return err
	}

	stageRoot := filepath.Join(s.updates, "runtime-"+time.Now().Format("20060102-150405"))
	cleanStaging(s.updates)
	_ = os.RemoveAll(stageRoot)
	if err := os.MkdirAll(stageRoot, 0o755); err != nil {
		s.fail(err)
		return err
	}
	stagingRuntime := filepath.Join(stageRoot, "runtime")
	if err := os.MkdirAll(stagingRuntime, 0o755); err != nil {
		s.fail(err)
		return err
	}
	if err := s.downloadRuntimePack(ctx, client, pack, deps, stageRoot, stagingRuntime, cfg.Update.Mirror); err != nil {
		s.fail(err)
		return err
	}
	writeRuntimeMarker(stagingRuntime, deps, pack, pack.Name)
	if err := writePending(s.updates, pendingUpdate{
		Kind:    "runtime",
		Staging: stagingRuntime,
		Version: deps.versionLabel(),
	}); err != nil {
		s.fail(err)
		return err
	}
	s.setStatus(Status{
		Stage:        StageReady,
		Target:       "runtime",
		Message:      "runtime dependencies are ready; restart to install",
		Version:      deps.versionLabel(),
		Progress:     100,
		NeedsRestart: true,
	})
	return nil
}
