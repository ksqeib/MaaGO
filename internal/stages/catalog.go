// Package stages builds the stage picker catalog from the installed MaaCore
// resource and, when available, MAA's StageActivityV2 activity snapshot.
package stages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ActivityCacheFile is the cached StageActivityV2.json snapshot.
const ActivityCacheFile = "StageActivityV2.json"

// ActivityURL is MAA's public stage/activity snapshot.
const ActivityURL = "https://api.maa.plus/MaaAssistantArknights/api/gui/StageActivityV2.json"

// Stage is one selectable stage in the picker.
type Stage struct {
	Code      string `json:"code"`
	Name      string `json:"name,omitempty"`
	AP        int    `json:"ap,omitempty"`
	Group     string `json:"group,omitempty"`
	OpenDays  []int  `json:"openDays,omitempty"`
	Start     string `json:"start,omitempty"`
	End       string `json:"end,omitempty"`
	Available bool   `json:"available"`
	Note      string `json:"note,omitempty"`
}

// Group is a labelled block inside a tab.
type Group struct {
	Label  string  `json:"label"`
	Stages []Stage `json:"stages"`
}

// Tab groups stages by kind.
type Tab struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Groups []Group `json:"groups"`
}

// Catalog is the payload returned to the frontend.
type Catalog struct {
	UpdatedAt string `json:"updatedAt"`
	Source    string `json:"source"`
	Tabs      []Tab  `json:"tabs"`
}

type stagesFileEntry struct {
	APCost int    `json:"apCost"`
	Code   string `json:"code"`
}

type activityClient struct {
	SideStoryStage     map[string]activityEntry `json:"sideStoryStage"`
	ResourceCollection activityInfo             `json:"resourceCollection"`
}

type activityEntry struct {
	Activity activityInfo    `json:"Activity"`
	Stages   []activityStage `json:"Stages"`
}

type activityInfo struct {
	Tip                  string `json:"Tip"`
	StageName            string `json:"StageName"`
	UtcStartTime         string `json:"UtcStartTime"`
	UtcExpireTime        string `json:"UtcExpireTime"`
	TimeZone             int    `json:"TimeZone"`
	IsResourceCollection bool   `json:"IsResourceCollection"`
}

type activityStage struct {
	Display string `json:"Display"`
	Value   string `json:"Value"`
	Drop    string `json:"Drop"`
}

type stageMeta struct {
	name string
	days []int
}

// CN resource stage families and their weekly schedule (0 = Sunday).
var resourceFamily = map[string]stageMeta{
	"CE": {name: "龙门币", days: []int{2, 4, 6, 0}},
	"LS": {name: "经验", days: nil},
	"AP": {name: "红票", days: []int{1, 4, 6, 0}},
	"CA": {name: "技能", days: []int{2, 3, 5, 0}},
	"SK": {name: "碳", days: []int{1, 3, 5, 6}},
}

var prMeta = map[string]stageMeta{
	"A": {name: "奶/盾芯片", days: []int{1, 4, 5, 0}},
	"B": {name: "术/狙芯片", days: []int{1, 2, 5, 6}},
	"C": {name: "先/辅芯片", days: []int{3, 4, 6, 0}},
	"D": {name: "近/特芯片", days: []int{2, 3, 6, 0}},
}

var (
	mainCodeRe   = regexp.MustCompile(`^\d+-\d+$`)
	sideCodeRe   = regexp.MustCompile(`^[HRS]\d+-\d+$`)
	resourceRe   = regexp.MustCompile(`^(CE|LS|AP|CA|SK)-\d+$`)
	prCodeRe     = regexp.MustCompile(`^PR-([A-D])-([12])$`)
	letterCodeRe = regexp.MustCompile(`^([A-Za-z]+)-`)
)

var tabOrder = []string{"main", "resource", "event", "annihilation", "other"}
var tabLabel = map[string]string{
	"main":         "主线",
	"resource":     "资源",
	"event":        "活动",
	"annihilation": "剿灭",
	"other":        "其他",
}

// Build assembles the picker catalog. cachePath may be empty when activity data
// is unavailable; the catalog still works from resource/stages.json.
func Build(coreDir, clientType, cachePath string, now time.Time) (Catalog, error) {
	if now.IsZero() {
		now = time.Now()
	}
	entries, err := readStages(coreDir)
	if err != nil {
		return Catalog{}, err
	}

	stages := make(map[string]Stage, len(entries)+32)
	for _, entry := range entries {
		code := strings.TrimSpace(entry.Code)
		if code == "" {
			continue
		}
		st := classify(code)
		st.AP = entry.APCost
		stages[code] = st
	}
	if _, ok := stages["Annihilation"]; !ok {
		stages["Annihilation"] = Stage{Code: "Annihilation", Name: "剿灭", Group: "剿灭"}
	}

	source := "resource"
	resourceOpen := false
	activity, activityErr := readActivity(cachePath, clientType)
	if activityErr == nil {
		source = "resource+activity"
		resourceOpen = activityActive(activity.ResourceCollection, now)
		for _, entry := range activity.SideStoryStage {
			info := entry.Activity
			group := strings.TrimSpace(info.StageName)
			if group == "" {
				group = strings.TrimSpace(info.Tip)
			}
			if group == "" {
				group = "活动"
			}
			start, end := parseWindow(info.UtcStartTime, info.UtcExpireTime, info.TimeZone)
			for _, raw := range entry.Stages {
				code := strings.TrimSpace(raw.Value)
				if code == "" {
					continue
				}
				st := stages[code]
				st.Code = code
				if st.Name == "" {
					st.Name = strings.TrimSpace(raw.Display)
				}
				// A stage can appear in several activities (rerun + original).
				// Prefer the currently running window, then the latest one.
				existingStart, _ := time.Parse(time.RFC3339, st.Start)
				existingEnd, _ := time.Parse(time.RFC3339, st.End)
				replace := st.Start == ""
				if !replace {
					existingActive := windowActive(existingStart, existingEnd, now)
					newActive := windowActive(start, end, now)
					switch {
					case newActive && !existingActive:
						replace = true
					case !newActive && !existingActive && start.After(existingStart):
						replace = true
					}
				}
				if replace {
					st.Group = group
					st.Start = ""
					st.End = ""
					if !start.IsZero() {
						st.Start = start.Format(time.RFC3339)
					}
					if !end.IsZero() {
						st.End = end.Format(time.RFC3339)
					}
				}
				stages[code] = st
			}
		}
	}

	for code, st := range stages {
		st.Code = code
		applyAvailability(&st, now, resourceOpen)
		stages[code] = st
	}

	catalog := Catalog{
		UpdatedAt: now.Format(time.RFC3339),
		Source:    source,
		Tabs:      buildTabs(stages),
	}
	return catalog, nil
}

func readStages(coreDir string) ([]stagesFileEntry, error) {
	path := filepath.Join(coreDir, "resource", "stages.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var entries []stagesFileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return entries, nil
}

func readActivity(cachePath, clientType string) (activityClient, error) {
	var result activityClient
	if strings.TrimSpace(cachePath) == "" {
		return result, os.ErrNotExist
	}
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return result, err
	}
	var all map[string]activityClient
	if err := json.Unmarshal(data, &all); err != nil {
		return result, err
	}
	if client, ok := all[clientType]; ok {
		return client, nil
	}
	if client, ok := all["Official"]; ok {
		return client, nil
	}
	return result, os.ErrNotExist
}

func classify(code string) Stage {
	switch {
	case code == "Annihilation":
		return Stage{Code: code, Name: "剿灭", Group: "剿灭"}
	case strings.HasPrefix(code, "PR-"):
		if match := prCodeRe.FindStringSubmatch(code); match != nil {
			meta := prMeta[match[1]]
			return Stage{Code: code, Name: meta.name, Group: "芯片", OpenDays: meta.days}
		}
		return Stage{Code: code, Group: "芯片"}
	case resourceRe.MatchString(code):
		match := resourceRe.FindStringSubmatch(code)
		meta := resourceFamily[match[1]]
		return Stage{Code: code, Name: meta.name, Group: "资源收集", OpenDays: meta.days}
	case mainCodeRe.MatchString(code):
		chapter := strings.SplitN(code, "-", 2)[0]
		number, _ := strconv.Atoi(chapter)
		group := "主线"
		if number > 0 {
			group = fmt.Sprintf("第 %d 章", number)
		}
		return Stage{Code: code, Group: group}
	case sideCodeRe.MatchString(code):
		return Stage{Code: code, Group: "主线"}
	default:
		if match := letterCodeRe.FindStringSubmatch(code); match != nil {
			return Stage{Code: code, Group: "活动 " + strings.ToUpper(match[1])}
		}
		return Stage{Code: code, Group: "其他"}
	}
}

func applyAvailability(stage *Stage, now time.Time, resourceOpen bool) {
	stage.Available = true
	stage.Note = ""
	switch {
	case stage.Code == "Annihilation":
		stage.Note = "每周作战"
	case stage.Start != "" || stage.End != "":
		start, _ := time.Parse(time.RFC3339, stage.Start)
		end, _ := time.Parse(time.RFC3339, stage.End)
		switch {
		case !start.IsZero() && now.Before(start):
			stage.Available = false
			stage.Note = "未开始"
		case !end.IsZero() && now.After(end):
			stage.Available = false
			stage.Note = "已结束"
		default:
			stage.Note = "进行中"
		}
	case len(stage.OpenDays) > 0:
		if resourceOpen {
			stage.Note = "资源全开放"
			break
		}
		today := int(now.Weekday())
		for _, day := range stage.OpenDays {
			if day == today {
				stage.Available = true
				stage.Note = "今日开放"
				return
			}
		}
		stage.Available = false
		stage.Note = openDaysText(stage.OpenDays)
	}
}

var weekdayNames = []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

func openDaysText(days []int) string {
	parts := make([]string, 0, len(days))
	for _, day := range days {
		if day >= 0 && day < len(weekdayNames) {
			parts = append(parts, weekdayNames[day])
		}
	}
	return strings.Join(parts, "/") + "开放"
}

func buildTabs(stages map[string]Stage) []Tab {
	grouped := make(map[string]map[string][]Stage)
	for _, st := range stages {
		tab := tabOf(st)
		if grouped[tab] == nil {
			grouped[tab] = make(map[string][]Stage)
		}
		grouped[tab][st.Group] = append(grouped[tab][st.Group], st)
	}

	tabs := make([]Tab, 0, len(tabOrder))
	for _, id := range tabOrder {
		groups, ok := grouped[id]
		if !ok {
			continue
		}
		tab := Tab{ID: id, Label: tabLabel[id]}
		labels := make([]string, 0, len(groups))
		for label := range groups {
			labels = append(labels, label)
		}
		sort.Slice(labels, func(i, j int) bool {
			return groupLess(id, labels[i], labels[j], groups[labels[i]], groups[labels[j]])
		})
		for _, label := range labels {
			list := groups[label]
			sort.Slice(list, func(i, j int) bool {
				return stageLess(list[i], list[j])
			})
			tab.Groups = append(tab.Groups, Group{Label: label, Stages: list})
		}
		tabs = append(tabs, tab)
	}
	return tabs
}

func tabOf(stage Stage) string {
	code := stage.Code
	switch {
	case code == "Annihilation":
		return "annihilation"
	case resourceRe.MatchString(code) || prCodeRe.MatchString(code):
		return "resource"
	case mainCodeRe.MatchString(code) || sideCodeRe.MatchString(code):
		return "main"
	case stage.Start != "" || stage.End != "":
		return "event"
	case letterCodeRe.MatchString(code):
		return "event"
	default:
		return "other"
	}
}

func groupLess(tab, a, b string, left, right []Stage) bool {
	switch tab {
	case "main":
		an, aok := chapterNumber(a)
		bn, bok := chapterNumber(b)
		if aok && bok && an != bn {
			return an < bn
		}
	}
	return a < b
}

func chapterNumber(label string) (int, bool) {
	label = strings.TrimSpace(label)
	label = strings.TrimPrefix(label, "第 ")
	label = strings.TrimSuffix(label, " 章")
	n, err := strconv.Atoi(label)
	return n, err == nil
}

func stageLess(a, b Stage) bool {
	an, aok := numericStage(a.Code)
	bn, bok := numericStage(b.Code)
	if aok && bok {
		if an.chapter != bn.chapter {
			return an.chapter < bn.chapter
		}
		if an.stage != bn.stage {
			return an.stage < bn.stage
		}
		return an.kind < bn.kind
	}
	return a.Code < b.Code
}

type numericCode struct {
	kind    int
	chapter int
	stage   int
}

func numericStage(code string) (numericCode, bool) {
	if mainCodeRe.MatchString(code) {
		parts := strings.SplitN(code, "-", 2)
		chapter, _ := strconv.Atoi(parts[0])
		stage, _ := strconv.Atoi(parts[1])
		return numericCode{kind: 0, chapter: chapter, stage: stage}, true
	}
	if sideCodeRe.MatchString(code) {
		kind := 0
		switch code[0] {
		case 'H':
			kind = 1
		case 'R':
			kind = 2
		case 'S':
			kind = 3
		}
		body := code[1:]
		parts := strings.SplitN(body, "-", 2)
		chapter, _ := strconv.Atoi(parts[0])
		stage, _ := strconv.Atoi(parts[1])
		return numericCode{kind: kind, chapter: chapter, stage: stage}, true
	}
	return numericCode{}, false
}

func activityActive(info activityInfo, now time.Time) bool {
	start, end := parseWindow(info.UtcStartTime, info.UtcExpireTime, info.TimeZone)
	return windowActive(start, end, now)
}

func windowActive(start, end, now time.Time) bool {
	if start.IsZero() && end.IsZero() {
		return false
	}
	if !start.IsZero() && now.Before(start) {
		return false
	}
	if !end.IsZero() && now.After(end) {
		return false
	}
	return true
}

func parseWindow(startText, endText string, zoneHours int) (time.Time, time.Time) {
	location := time.FixedZone("MAA", zoneHours*3600)
	parse := func(value string) time.Time {
		value = strings.TrimSpace(value)
		if value == "" {
			return time.Time{}
		}
		for _, layout := range []string{"2006/01/02 15:04:05", "2006-01-02 15:04:05"} {
			if parsed, err := time.ParseInLocation(layout, value, location); err == nil {
				return parsed
			}
		}
		return time.Time{}
	}
	return parse(startText), parse(endText)
}
