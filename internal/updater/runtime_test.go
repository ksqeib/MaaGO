package updater

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testDeps() coreDependencies {
	return coreDependencies{
		Dependencies: map[string]coreDependency{
			"opencv":      {Version: "4.12.0"},
			"onnxruntime": {Version: "1.19.2", Flavor: "directml"},
			"directml":    {Version: "1.15.4"},
			"fastdeploy":  {Version: "1.0.7"},
		},
		RuntimeFiles: map[string]string{
			"opencv_world4_maa.dll":    "opencv",
			"onnxruntime_maa.dll":      "onnxruntime",
			"DirectML.dll":             "directml",
			"fastdeploy_ppocr_maa.dll": "fastdeploy",
		},
	}
}

func testV2Deps() coreDependencies {
	return coreDependencies{
		SchemaVersion: 2,
		Runtime: map[string]runtimePlatform{
			"windows-x64": {
				Bundle: runtimeBundle{
					URL:    "https://github.com/ksqeib/MaaCoreBuild/releases/download/runtime-v1/MaaCore-runtime-windows-x64-v1.zip",
					SHA256: "e6f040a80973bb2149d87aa2b428130c1a9be77ff5f571bb0b08927a6c900f9e",
					Size:   38417030,
				},
				Files: map[string]runtimeFileSpec{
					"opencv_world4_maa.dll": {
						Dependency: "opencv",
						Version:    "4.12.0",
						SHA256:     "0455560c75bc456b5f758c94af653b6f0397ed8a28b9605cf52c62e7c8f18781",
					},
					"onnxruntime_maa.dll": {
						Dependency: "onnxruntime",
						Version:    "1.19.2",
						Flavor:     "directml",
						SHA256:     "410de3464172dae2945a4663f274e34c59965bc8233b0411de16baab56dda1a7",
					},
					"DirectML.dll": {
						Dependency: "directml",
						Version:    "1.15.4",
						SHA256:     "9c9e6d822561c6c41b90e6994b3e8857cf1d66dbfb1e0c4c799c7c89b4e92da1",
					},
					"fastdeploy_ppocr_maa.dll": {
						Dependency: "fastdeploy",
						Version:    "1.0.7",
						SHA256:     "52591c163ed7d6b8c0196a54ca5168fc1eb3b785f06253e200424771e9183fb6",
					},
				},
			},
		},
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestMatchRuntimePack(t *testing.T) {
	if pack := matchRuntimePack(testDeps()); pack == nil {
		t.Fatal("expected the known v0.1.3 pack to match deps.json v0.1.4")
	}
	deps := testDeps()
	deps.Dependencies["opencv"] = coreDependency{Version: "9.9.9"}
	if pack := matchRuntimePack(deps); pack != nil {
		t.Fatalf("unexpected pack for opencv 9.9.9: %s", pack.Name)
	}
	deps = testDeps()
	deps.Dependencies["onnxruntime"] = coreDependency{Version: "1.19.2", Flavor: "cpu"}
	if pack := matchRuntimePack(deps); pack != nil {
		t.Fatal("flavor mismatch must not match the DirectML pack")
	}
}

func TestV2RuntimeBundle(t *testing.T) {
	deps := coreDependencies{
		SchemaVersion: 2,
		Runtime: map[string]runtimePlatform{
			"windows-x64": {
				Bundle: runtimeBundle{
					URL:    "https://example.com/runtime.zip",
					SHA256: "abc",
					Size:   123,
				},
				Files: map[string]runtimeFileSpec{
					"opencv_world4_maa.dll": {Dependency: "opencv", Version: "4.12.0", SHA256: "a"},
					"onnxruntime_maa.dll":   {Dependency: "onnxruntime", Version: "1.19.2", Flavor: "directml", SHA256: "b"},
				},
			},
		},
	}
	pack := runtimePackFor(deps)
	if pack == nil || pack.URL != "https://example.com/runtime.zip" || pack.SHA256 != "abc" || pack.Size != 123 {
		t.Fatalf("unexpected v2 pack: %+v", pack)
	}
	if pack.Files["onnxruntime_maa.dll"].ArchivePath != "onnxruntime_maa.dll" {
		t.Fatal("v2 runtime bundle files must be stored at the archive root")
	}
	if pack.Files["onnxruntime_maa.dll"].Version != "1.19.2" {
		t.Fatal("v2 per-file version was not parsed")
	}
	if pack.Dependencies["onnxruntime"] != "1.19.2" || pack.Flavors["onnxruntime"] != "directml" {
		t.Fatalf("v2 dependency metadata: %+v %+v", pack.Dependencies, pack.Flavors)
	}
}

func TestRuntimeStatusReportsMissingFiles(t *testing.T) {
	core := t.TempDir()
	data := `{"dependencies":{"opencv":{"version":"4.12.0"},"onnxruntime":{"version":"1.19.2","flavor":"directml"},"directml":{"version":"1.15.4"},"fastdeploy":{"version":"1.0.7"}},"runtimeFiles":{"opencv_world4_maa.dll":"opencv","onnxruntime_maa.dll":"onnxruntime","DirectML.dll":"directml","fastdeploy_ppocr_maa.dll":"fastdeploy"}}`
	if err := os.WriteFile(filepath.Join(core, "deps.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	status := New(core, nil).RuntimeStatus()
	if status.OK {
		t.Fatal("status should report missing runtime files")
	}
	if !status.Available || !status.CanRepair {
		t.Fatalf("unexpected status: %+v", status)
	}
	if len(status.Missing) != 4 {
		t.Fatalf("missing=%v", status.Missing)
	}
}

func TestRuntimeStatusInstalledCore(t *testing.T) {
	core := os.Getenv("MAAGO_TEST_CORE_DIR")
	if core == "" {
		t.Skip("set MAAGO_TEST_CORE_DIR to inspect an installed core")
	}
	status := New(core, nil).RuntimeStatus()
	t.Logf("status=%+v", status)
	if !status.OK {
		t.Fatal(status.Message)
	}
}

func TestApplyPendingRuntimeCopiesFiles(t *testing.T) {
	root := t.TempDir()
	core := filepath.Join(root, "core")
	updates := filepath.Join(root, ".maago-updates")
	if err := os.MkdirAll(core, 0o755); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(updates, "runtime-1")
	writeFile(t, filepath.Join(staging, ".maago-runtime.json"), "{}")
	writeFile(t, filepath.Join(staging, "opencv_world4_maa.dll"), "new")
	if err := writePending(updates, pendingUpdate{Kind: "runtime", Staging: staging, Version: "test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPending(core); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(core, "opencv_world4_maa.dll"))
	if err != nil || string(data) != "new" {
		t.Fatalf("runtime file: %q err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(core, ".maago-runtime.json")); err != nil {
		t.Fatal("runtime marker missing:", err)
	}
}

func TestDownloadRuntimeStages(t *testing.T) {
	if os.Getenv("MAAGO_TEST_NETWORK") == "" {
		t.Skip("set MAAGO_TEST_NETWORK=1 to download the runtime pack")
	}
	root := t.TempDir()
	core := filepath.Join(root, "core")
	if err := os.MkdirAll(core, 0o755); err != nil {
		t.Fatal(err)
	}
	data := mustJSON(t, testV2Deps())
	if err := os.WriteFile(filepath.Join(core, "deps.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	service := New(core, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	if err := service.DownloadRuntime(ctx); err != nil {
		t.Fatal(err)
	}
	pending, ok, err := readPending(filepath.Join(root, ".maago-updates"))
	if err != nil || !ok {
		t.Fatalf("pending: ok=%v err=%v", ok, err)
	}
	for _, name := range []string{"opencv_world4_maa.dll", "onnxruntime_maa.dll", "DirectML.dll", "fastdeploy_ppocr_maa.dll", ".maago-runtime.json"} {
		if _, err := os.Stat(filepath.Join(pending.Staging, name)); err != nil {
			t.Fatalf("staged runtime is missing %s: %v", name, err)
		}
	}
	if _, err := ApplyPending(core); err != nil {
		t.Fatal(err)
	}
	status := service.RuntimeStatus()
	if !status.OK {
		t.Fatalf("runtime status after apply: %+v", status)
	}
}

func TestDownloadCorePreservesRuntime(t *testing.T) {
	source := os.Getenv("MAAGO_TEST_CORE_SOURCE")
	if source == "" || os.Getenv("MAAGO_TEST_NETWORK") == "" {
		t.Skip("set MAAGO_TEST_NETWORK=1 and MAAGO_TEST_CORE_SOURCE to an installed core")
	}
	root := t.TempDir()
	core := filepath.Join(root, "core")
	if err := copyPath(source, core); err != nil {
		t.Fatal(err)
	}
	service := New(core, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	version := os.Getenv("MAAGO_TEST_CORE_VERSION")
	if version == "" {
		version = "0.1.5"
	}
	if err := service.DownloadCore(ctx, version); err != nil {
		t.Fatal(err)
	}
	pending, ok, err := readPending(filepath.Join(root, ".maago-updates"))
	if err != nil || !ok {
		t.Fatalf("pending: ok=%v err=%v", ok, err)
	}
	for _, name := range []string{"MaaCore.dll", "deps.json", ".maago-runtime.json", "opencv_world4_maa.dll", "onnxruntime_maa.dll", "DirectML.dll", "fastdeploy_ppocr_maa.dll", "MaaWin32ControlUnit.dll"} {
		if _, err := os.Stat(filepath.Join(pending.Staging, name)); err != nil {
			t.Fatalf("staged core is missing %s: %v", name, err)
		}
	}
}

func TestExtractRuntimeArchive(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "pack.zip")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	contents := map[string]string{
		"bin/a.dll": "hello",
		"bin/b.dll": "world",
	}
	for name, data := range contents {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	file.Close()

	sum := func(s string) (string, int64) {
		hash := sha256.Sum256([]byte(s))
		return hex.EncodeToString(hash[:]), int64(len(s))
	}
	aSHA, aSize := sum("hello")
	bSHA, bSize := sum("world")
	pack := &runtimePack{Files: map[string]runtimePackFile{
		"a.dll": {Dependency: "a", ArchivePath: "bin/a.dll", Size: aSize, SHA256: aSHA},
		"b.dll": {Dependency: "b", ArchivePath: "bin/b.dll", Size: bSize, SHA256: bSHA},
	}}
	deps := coreDependencies{
		Dependencies: map[string]coreDependency{"a": {Version: "1"}, "b": {Version: "2"}},
		RuntimeFiles: map[string]string{"a.dll": "a", "b.dll": "b"},
	}
	dest := filepath.Join(root, "out")
	if err := extractRuntimeArchive(archive, dest, deps, pack); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"a.dll": "hello", "b.dll": "world"} {
		data, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != want {
			t.Fatalf("%s=%q", name, data)
		}
	}
}
