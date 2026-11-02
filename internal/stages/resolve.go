package stages

import (
	"fmt"
	"strings"
	"time"
)

// Ordered returns the user's ordered stage list. It accepts the new "stages"
// array and falls back to the legacy single "stage" option.
func Ordered(options map[string]any) []string {
	if options == nil {
		return nil
	}
	var result []string
	switch value := options["stages"].(type) {
	case []string:
		result = append(result, value...)
	case []any:
		for _, item := range value {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
	}
	if len(result) == 0 {
		if legacy, ok := options["stage"].(string); ok && strings.TrimSpace(legacy) != "" {
			result = append(result, strings.TrimSpace(legacy))
		}
	}
	return result
}

// Resolve picks the first currently available stage from the ordered list.
// Unknown stages are treated as available so manually typed codes still work.
// The returned note is meant for the run log.
func Resolve(coreDir, clientType, cachePath string, options map[string]any, now time.Time) (string, string) {
	ordered := Ordered(options)
	if len(ordered) == 0 {
		return "", ""
	}
	first := strings.TrimSpace(ordered[0])
	if first == "" {
		return "", ""
	}

	catalog, err := Build(coreDir, clientType, cachePath, now)
	if err != nil {
		return first, ""
	}
	available := make(map[string]Stage)
	for _, tab := range catalog.Tabs {
		for _, group := range tab.Groups {
			for _, stage := range group.Stages {
				available[stage.Code] = stage
			}
		}
	}

	var skipped []string
	for _, raw := range ordered {
		code := strings.TrimSpace(raw)
		if code == "" {
			return "", "使用当前/上次关卡"
		}
		stage, known := available[code]
		if !known || stage.Available {
			note := ""
			if len(skipped) > 0 {
				note = "跳过不可用关卡：" + strings.Join(skipped, "、")
			}
			return code, note
		}
		label := code
		if stage.Note != "" {
			label += "（" + stage.Note + "）"
		}
		skipped = append(skipped, label)
	}

	return first, fmt.Sprintf("已选关卡当前都不可用，按顺序使用 %s", first)
}
