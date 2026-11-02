package maa

import (
	"encoding/json"
	"math"
	"testing"

	"MaaGO/internal/config"
)

func TestBuildFightParamsExpiringOnly(t *testing.T) {
	cfg := config.Default()
	task := config.TaskConfig{
		Type: config.TaskFight,
		Options: map[string]any{
			"stage":              "1-7",
			"times":              0,
			"medicine":           false,
			"medicineCount":      3,
			"expiringMedicine":   true,
			"medicineExpireDays": 1,
			"stone":              false,
			"stoneCount":         2,
			"series":             "1",
		},
	}
	raw, err := BuildTaskParams(task, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		t.Fatal(err)
	}
	if got := params["medicine"]; got != float64(0) {
		t.Fatalf("medicine = %v, want 0", got)
	}
	if got := params["medicine_expire_days"]; got != float64(1) {
		t.Fatalf("medicine_expire_days = %v, want 1", got)
	}
	if got := params["stone"]; got != float64(0) {
		t.Fatalf("stone = %v, want 0", got)
	}
	if got := params["times"]; got != float64(math.MaxInt32) {
		t.Fatalf("times = %v, want MaxInt32", got)
	}

	task.Options["medicine"] = true
	task.Options["stone"] = true
	raw, err = BuildTaskParams(task, cfg)
	if err != nil {
		t.Fatal(err)
	}
	params = map[string]any{}
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		t.Fatal(err)
	}
	if got := params["medicine"]; got != float64(3) {
		t.Fatalf("medicine = %v, want 3", got)
	}
	if got := params["stone"]; got != float64(2) {
		t.Fatalf("stone = %v, want 2", got)
	}
}

func TestBuildFightParamsClampsExpiryWindow(t *testing.T) {
	cfg := config.Default()
	task := config.TaskConfig{
		Type: config.TaskFight,
		Options: map[string]any{
			"expiringMedicine":   true,
			"medicineExpireDays": 0,
			"series":             "1",
		},
	}
	raw, err := BuildTaskParams(task, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		t.Fatal(err)
	}
	if got := params["medicine_expire_days"]; got != float64(1) {
		t.Fatalf("medicine_expire_days = %v, want clamped 1", got)
	}
}
