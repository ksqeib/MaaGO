package stages

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFixture(t *testing.T) (coreDir, cachePath string) {
	t.Helper()
	coreDir = t.TempDir()
	resourceDir := filepath.Join(coreDir, "resource")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stagesJSON := `[
		{"code":"1-7","apCost":6},
		{"code":"CE-6","apCost":36},
		{"code":"PR-A-1","apCost":18},
		{"code":"TL-9","apCost":21},
		{"code":"GT-1","apCost":9},
		{"code":"Annihilation","apCost":0}
	]`
	if err := os.WriteFile(filepath.Join(resourceDir, "stages.json"), []byte(stagesJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cachePath = filepath.Join(t.TempDir(), ActivityCacheFile)
	activityJSON := `{
		"Official": {
			"sideStoryStage": {
				"TL": {
					"Activity": {
						"Tip": "SideStory「时间啊，向前！」",
						"StageName": "时间啊，向前！",
						"UtcStartTime": "2026/11/01 12:00:00",
						"UtcExpireTime": "2026/11/22 03:59:59",
						"TimeZone": 8
					},
					"Stages": [
						{"Display":"TL-9","Value":"TL-9","Drop":"31113"}
					]
				},
				"OLD": {
					"Activity": {
						"Tip": "过期活动",
						"StageName": "过期活动",
						"UtcStartTime": "2026/08/01 12:00:00",
						"UtcExpireTime": "2026/08/10 03:59:59",
						"TimeZone": 8
					},
					"Stages": [
						{"Display":"GT-1","Value":"GT-1","Drop":"30012"}
					]
				}
			}
		}
	}`
	if err := os.WriteFile(cachePath, []byte(activityJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return coreDir, cachePath
}

func TestBuildCatalog(t *testing.T) {
	coreDir, cachePath := writeFixture(t)
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.Local) // Wednesday
	catalog, err := Build(coreDir, "Official", cachePath, now)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Source != "resource+activity" {
		t.Fatalf("source = %q", catalog.Source)
	}
	index := map[string]Stage{}
	for _, tab := range catalog.Tabs {
		for _, group := range tab.Groups {
			for _, stage := range group.Stages {
				index[stage.Code] = stage
			}
		}
	}
	if !index["TL-9"].Available || index["TL-9"].Group != "时间啊，向前！" {
		t.Fatalf("TL-9 = %+v", index["TL-9"])
	}
	if index["GT-1"].Available {
		t.Fatalf("GT-1 should be expired: %+v", index["GT-1"])
	}
	if index["CE-6"].Available {
		t.Fatalf("CE-6 should be closed on Wednesday: %+v", index["CE-6"])
	}
	if index["1-7"].Group != "第 1 章" {
		t.Fatalf("1-7 group = %q", index["1-7"].Group)
	}
}

func TestResolveOrderedStages(t *testing.T) {
	coreDir, cachePath := writeFixture(t)
	options := map[string]any{"stages": []any{"CE-6", "TL-9", "1-7"}}

	wednesday := time.Date(2026, 11, 4, 10, 0, 0, 0, time.Local)
	code, note := Resolve(coreDir, "Official", cachePath, options, wednesday)
	if code != "TL-9" {
		t.Fatalf("Wednesday resolved %q (%s), want TL-9", code, note)
	}

	tuesday := time.Date(2026, 11, 3, 10, 0, 0, 0, time.Local)
	code, _ = Resolve(coreDir, "Official", cachePath, options, tuesday)
	if code != "CE-6" {
		t.Fatalf("Tuesday resolved %q, want CE-6", code)
	}
}

func TestResolveLegacyStage(t *testing.T) {
	coreDir, cachePath := writeFixture(t)
	code, _ := Resolve(coreDir, "Official", cachePath, map[string]any{"stage": "1-7"}, time.Now())
	if code != "1-7" {
		t.Fatalf("legacy stage resolved %q", code)
	}
}

func TestBuildInstalledCore(t *testing.T) {
	coreDir, err := filepath.Abs(filepath.Join("..", "..", "core"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(coreDir, "resource", "stages.json")); err != nil {
		t.Skip("installed core resource is not available")
	}
	cachePath := filepath.Join(t.TempDir(), ActivityCacheFile)
	if appData := os.Getenv("APPDATA"); appData != "" {
		candidate := filepath.Join(appData, "MaaGO", "cache", ActivityCacheFile)
		if _, err := os.Stat(candidate); err == nil {
			cachePath = candidate
		}
	}
	catalog, err := Build(coreDir, "Official", cachePath, time.Now())
	t.Logf("source=%s", catalog.Source)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, tab := range catalog.Tabs {
		count := 0
		for _, group := range tab.Groups {
			count += len(group.Stages)
		}
		total += count
		t.Logf("tab %s: %d groups, %d stages", tab.ID, len(tab.Groups), count)
	}
	if total == 0 {
		t.Fatal("catalog is empty")
	}
}
