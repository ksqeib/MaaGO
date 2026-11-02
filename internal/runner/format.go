package runner

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var facilityNames = map[string]string{
	"Mfg":       "制造站",
	"Trade":     "贸易站",
	"Power":     "发电站",
	"Control":   "控制中枢",
	"Reception": "会客室",
	"Office":    "办公室",
	"Dorm":      "宿舍",
	"Training":  "训练室",
	"Workshop":  "加工站",
	"Meeting":   "会客室",
}

var productNames = map[string]string{
	"Drone":         "无人机",
	"PureGold":      "赤金",
	"OriginStone":   "源石碎片",
	"CombatRecord":  "作战记录",
	"SyntheticJade": "合成玉",
	"Money":         "龙门币",
	"HR":            "人力资源",
	"General":       "综合",
	"MoodAddition":  "心情加成",
}

var limitTaskNames = map[string]string{
	"StartButton1":            "开战次数",
	"StartButton2":            "开战次数",
	"MedicineConfirm":         "理智药",
	"ExpiringMedicineConfirm": "临期理智药",
	"StoneConfirm":            "源石",
	"CloseStonePage":          "碎石页面",
	"ReturnButton":            "返回按钮",
	"GoLastBattle":            "上次作战",
}

// formatExtra turns a MaaCore extra-info payload into a readable Chinese line.
// It returns "" when the payload cannot be parsed; callers then fall back to the
// original JSON.
func formatExtra(what string, raw json.RawMessage) string {
	var payload map[string]any
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &payload) != nil {
		return ""
	}
	switch what {
	case "ProductOfFacility":
		return fmt.Sprintf("%s 当前产品：%s", facilityText(payload), productText(str(payload, "product")))
	case "EnterFacility":
		return "进入设施：" + facilityText(payload)
	case "InfrastConfirmButton":
		return "确认设施：" + facilityText(payload)
	case "RecruitTagsDetected":
		return "公招标签：" + strings.Join(stringsOf(payload["tags"]), " / ")
	case "RecruitTagsSelected":
		return "已选标签：" + strings.Join(stringsOf(payload["tags"]), " + ")
	case "RecruitTagsRefreshed":
		return fmt.Sprintf("公招刷新：第 %d 次（上限 %d 次）", integer(payload, "count"), integer(payload, "refresh_limit"))
	case "RecruitResult":
		return formatRecruitResult(payload)
	case "FightTimes":
		cost := integer(payload, "sanity_cost")
		finished := integer(payload, "times_finished")
		series := integer(payload, "series")
		seriesText := ""
		if series > 0 {
			seriesText = fmt.Sprintf("，本次连战 %d 次", series)
		}
		if boolean(payload["finished"]) {
			return fmt.Sprintf("作战完成：已打 %d 次，每次 %d 理智%s", finished, cost, seriesText)
		}
		return fmt.Sprintf("作战进行：已打 %d 次，每次 %d 理智%s", finished, cost, seriesText)
	case "SanityBeforeStage":
		return fmt.Sprintf("当前理智：%d / %d", integer(payload, "current_sanity"), integer(payload, "max_sanity"))
	case "ExceededLimit":
		task := str(payload, "task")
		if label := limitTaskNames[task]; label != "" {
			task = label
		}
		return fmt.Sprintf("达到上限：%s（%d / %d）", task, integer(payload, "exec_times"), integer(payload, "max_times"))
	case "StageDrops":
		return formatStageDrops(payload)
	case "UseMedicine":
		return formatMedicine(payload)
	case "StageQueueUnableToAgent":
		return "关卡 " + str(payload, "stage_code") + " 无法代理"
	case "StageQueueMissionCompleted":
		return fmt.Sprintf("关卡 %s 完成 %d★", str(payload, "stage_code"), integer(payload, "stars"))
	default:
		text := formatJSONValue(payload, 0)
		if text == "" {
			return ""
		}
		if what != "" {
			return what + "：" + text
		}
		return text
	}
}

func formatRecruitResult(payload map[string]any) string {
	results := objectsOf(payload["result"])
	if len(results) == 0 {
		return ""
	}
	shown := results
	if len(shown) > 3 {
		shown = shown[:3]
	}
	parts := make([]string, 0, len(shown))
	for _, result := range shown {
		segment := fmt.Sprintf("%d★", integer(result, "level"))
		if tags := stringsOf(result["tags"]); len(tags) > 0 {
			segment += " " + strings.Join(tags, "+")
		}
		opers := objectsOf(result["opers"])
		if len(opers) > 0 {
			names := make([]string, 0, 4)
			for i, oper := range opers {
				if i >= 4 {
					break
				}
				if name := str(oper, "name"); name != "" {
					names = append(names, name)
				}
			}
			if len(names) > 0 {
				segment += " → " + strings.Join(names, "、")
				if len(opers) > 4 {
					segment += fmt.Sprintf(" 等%d人", len(opers))
				}
			}
		}
		parts = append(parts, segment)
	}
	text := "公招结果：" + strings.Join(parts, "；")
	if len(results) > len(shown) {
		text += fmt.Sprintf(" 等%d组", len(results))
	}
	return text
}

func formatStageDrops(payload map[string]any) string {
	stage := ""
	if stageMap, ok := payload["stage"].(map[string]any); ok {
		stage = str(stageMap, "stageCode")
		if stage == "" {
			stage = str(stageMap, "stageId")
		}
	}
	items := make([]string, 0, 6)
	total := 0
	for _, drop := range objectsOf(payload["drops"]) {
		if str(drop, "dropType") == "FURNITURE" || str(drop, "itemName") == "furni" {
			continue
		}
		name := str(drop, "itemName")
		if name == "" {
			name = str(drop, "itemId")
		}
		total++
		if len(items) >= 6 {
			continue
		}
		items = append(items, fmt.Sprintf("%s×%d", name, integer(drop, "quantity")))
	}
	if len(items) == 0 {
		return ""
	}
	text := "关卡掉落"
	if stage != "" {
		text = "关卡 " + stage + " 掉落"
	}
	text += "：" + strings.Join(items, "、")
	if total > len(items) {
		text += fmt.Sprintf(" 等%d项", total)
	}
	if stars := integer(payload, "stars"); stars > 0 {
		text += fmt.Sprintf("（%d★）", stars)
	}
	return text
}

func formatMedicine(payload map[string]any) string {
	count := integer(payload, "count")
	text := fmt.Sprintf("使用理智药：%d 瓶", count)
	if boolean(payload["is_expiring"]) {
		text += "（临期）"
	}
	medicines := objectsOf(payload["medicines"])
	if len(medicines) == 1 {
		medicine := medicines[0]
		if expire := integer(medicine, "expire_days"); expire >= 0 {
			text += fmt.Sprintf("，剩余 %d 天", expire)
		}
		if inventory := integer(medicine, "inventory"); inventory > 0 {
			text += fmt.Sprintf("，库存 %d", inventory)
		}
	}
	return text
}

func facilityText(payload map[string]any) string {
	facility := str(payload, "facility")
	if name := facilityNames[facility]; name != "" {
		facility = name
	}
	if facility == "" {
		facility = "设施"
	}
	index := integer(payload, "index")
	return fmt.Sprintf("%s #%d", facility, index+1)
}

func productText(product string) string {
	if name := productNames[product]; name != "" {
		return name
	}
	if product == "" {
		return "未知"
	}
	return product
}

func str(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}

func integer(payload map[string]any, key string) int {
	switch value := payload[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case string:
		number, _ := strconv.Atoi(value)
		return number
	}
	return 0
}

func boolean(value any) bool {
	result, _ := value.(bool)
	return result
}

func stringsOf(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func objectsOf(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if object, ok := item.(map[string]any); ok {
			result = append(result, object)
		}
	}
	return result
}

// formatConnection renders ConnectionInfo payloads.
func formatConnection(what string, raw json.RawMessage, why string) string {
	var payload map[string]any
	_ = json.Unmarshal(raw, &payload)
	switch what {
	case "UuidGot":
		return "设备 UUID：" + str(payload, "uuid")
	case "ResolutionGot":
		return fmt.Sprintf("设备分辨率：%d×%d", integer(payload, "width"), integer(payload, "height"))
	case "Connected":
		address := str(payload, "address")
		config := str(payload, "config")
		text := "设备已连接"
		if address != "" {
			text += "：" + address
		}
		if config != "" {
			text += "（配置 " + config + "）"
		}
		return text
	case "ConnectFailed":
		text := "设备连接失败"
		if why != "" {
			text += "：" + why
		}
		return text
	case "Reconnecting":
		return "正在重新连接设备"
	case "Reconnected":
		return "设备重新连接成功"
	case "Disconnect":
		text := "设备连接已断开"
		if why != "" {
			text += "：" + why
		}
		return text
	case "FastestWayToScreencap":
		return fmt.Sprintf("截图方式：%s（%dms）", str(payload, "method"), integer(payload, "cost"))
	case "ScreencapCost":
		text := fmt.Sprintf("截图耗时：平均 %dms，最大 %dms，最小 %dms", integer(payload, "avg"), integer(payload, "max"), integer(payload, "min"))
		if faults := integer(payload, "fault_times"); faults > 0 {
			text += fmt.Sprintf("，失败 %d 次", faults)
		}
		return text
	case "EmulatorFPS":
		return fmt.Sprintf("模拟器帧率：%d FPS", integer(payload, "fps"))
	case "ResolutionInfo":
		return fmt.Sprintf("分辨率信息：%d×%d", integer(payload, "width"), integer(payload, "height"))
	default:
		return formatJSONValue(payload, 0)
	}
}

var keyLabels = map[string]string{
	"facility": "设施", "index": "序号", "product": "产品", "tags": "标签",
	"level": "等级", "result": "结果", "opers": "干员", "id": "ID", "name": "名称",
	"drops": "掉落", "itemId": "物品ID", "itemName": "物品", "quantity": "数量",
	"dropType": "掉落类型", "stage": "关卡", "stageCode": "关卡", "stageId": "关卡ID",
	"stars": "星级", "current_sanity": "当前理智", "max_sanity": "理智上限",
	"sanity_cost": "每次理智", "times_finished": "已完成次数", "finished": "已完成",
	"exec_times": "已执行", "max_times": "上限", "task": "任务", "count": "数量",
	"is_expiring": "临期", "medicines": "药品", "expire_days": "剩余天数",
	"inventory": "库存", "use": "使用", "stats": "统计", "addQuantity": "增加数量",
	"height": "高度", "width": "宽度", "uuid": "UUID", "address": "地址",
	"config": "配置", "adb": "ADB", "cost": "耗时", "method": "方式",
	"alternatives": "候选方式", "avg": "平均", "max": "最大", "min": "最小",
	"fault_times": "失败次数", "fps": "帧率", "refresh_period_ns": "刷新周期",
	"why": "原因", "error": "错误", "msg": "消息", "message": "消息",
}

var keyOrder = []string{
	"facility", "index", "product", "task", "tags", "level", "result", "opers",
	"stage", "stageCode", "stageId", "stars", "drops", "itemName", "quantity",
	"current_sanity", "max_sanity", "sanity_cost", "times_finished", "finished",
	"count", "is_expiring", "medicines", "expire_days", "inventory", "use",
	"width", "height", "fps", "avg", "max", "min", "cost", "method",
}

func formatJSONValue(value any, depth int) string {
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			return ""
		}
		keys := orderedKeys(typed)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			rendered := formatJSONValue(typed[key], depth+1)
			if rendered == "" {
				continue
			}
			parts = append(parts, labelFor(key)+"："+rendered)
		}
		return strings.Join(parts, "，")
	case []any:
		if len(typed) == 0 {
			return ""
		}
		parts := make([]string, 0, len(typed))
		objectItems := 0
		for _, item := range typed {
			rendered := formatJSONValue(item, depth+1)
			if rendered == "" {
				continue
			}
			if _, ok := item.(map[string]any); ok {
				objectItems++
			}
			parts = append(parts, rendered)
		}
		separator := "、"
		if objectItems > 0 {
			separator = "；"
		}
		return strings.Join(parts, separator)
	case string:
		return typed
	case bool:
		if typed {
			return "是"
		}
		return "否"
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func orderedKeys(payload map[string]any) []string {
	seen := make(map[string]bool, len(payload))
	keys := make([]string, 0, len(payload))
	for _, key := range keyOrder {
		if _, ok := payload[key]; ok {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	rest := make([]string, 0)
	for key := range payload {
		if !seen[key] {
			rest = append(rest, key)
		}
	}
	sort.Strings(rest)
	return append(keys, rest...)
}

func labelFor(key string) string {
	if label := keyLabels[key]; label != "" {
		return label
	}
	return key
}
