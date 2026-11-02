package runner

import (
	"fmt"
	"time"

	"MaaGO/internal/config"
)

type step struct {
	Delay   time.Duration
	Level   string
	Message string
}

func scriptFor(task config.TaskConfig) []step {
	switch task.Type {
	case config.TaskStartUp:
		return startupScript(task.Options)
	case config.TaskFight:
		return fightScript(task.Options)
	case config.TaskInfrast:
		return infrastScript(task.Options)
	case config.TaskRecruit:
		return recruitScript(task.Options)
	case config.TaskMall:
		return mallScript(task.Options)
	case config.TaskAward:
		return awardScript(task.Options)
	default:
		return []step{{Delay: 600 * time.Millisecond, Message: "演示任务执行中…"}}
	}
}

func startupScript(options map[string]any) []step {
	client := clientTypeName(optString(options, "clientType", "Official"))
	steps := []step{
		{400 * time.Millisecond, "info", "检查 ADB 连接状态…"},
		{500 * time.Millisecond, "warn", "演示：跳过真实连接检查"},
		{400 * time.Millisecond, "info", fmt.Sprintf("客户端类型：%s", client)},
	}
	if optBool(options, "launchClient", true) {
		steps = append(steps,
			step{500 * time.Millisecond, "info", "启动明日方舟…"},
			step{700 * time.Millisecond, "info", "等待游戏加载…"},
			step{500 * time.Millisecond, "success", "已进入游戏主界面"},
		)
	}
	if account := optString(options, "accountSwitch", ""); account != "" {
		steps = append(steps, step{400 * time.Millisecond, "info", fmt.Sprintf("切换账号：%s", account)})
	}
	steps = append(steps, step{400 * time.Millisecond, "success", "开始唤醒完成"})
	return steps
}

func fightScript(options map[string]any) []step {
	stage := optString(options, "stage", "")
	if stage == "" {
		stage = "当前/上次"
	}
	times := optInt(options, "times", 0)
	fightCount := "直到理智耗尽"
	if times > 0 {
		fightCount = fmt.Sprintf("%d 次", times)
	}
	steps := []step{
		{400 * time.Millisecond, "info", fmt.Sprintf("关卡：%s", stage)},
		{400 * time.Millisecond, "info", "识别当前理智…"},
		{400 * time.Millisecond, "info", "理智：135 / 135"},
	}
	if optBool(options, "expiringMedicine", false) {
		days := optInt(options, "medicineExpireDays", 1)
		steps = append(steps, step{300 * time.Millisecond, "info", fmt.Sprintf("只使用 %d 天内过期的理智药", days)})
	}
	if optBool(options, "medicine", false) {
		steps = append(steps, step{300 * time.Millisecond, "info", "理智不足时使用普通理智药"})
	} else {
		steps = append(steps, step{300 * time.Millisecond, "info", "不会使用普通理智药"})
	}
	if optBool(options, "stone", false) {
		steps = append(steps, step{300 * time.Millisecond, "warn", "已启用源石补充理智"})
	}
	steps = append(steps,
		step{500 * time.Millisecond, "info", fmt.Sprintf("开始代理指挥，计划刷取 %s", fightCount)},
		step{700 * time.Millisecond, "info", "第 1 次作战完成"},
		step{600 * time.Millisecond, "info", "第 2 次作战完成"},
		step{600 * time.Millisecond, "success", "理智作战完成"},
	)
	return steps
}

func infrastScript(options map[string]any) []step {
	mode := optString(options, "mode", "default")
	modeName := map[string]string{
		"default":  "常规模式",
		"rotation": "队列轮换",
		"custom":   "自定义排班",
	}[mode]
	if modeName == "" {
		modeName = mode
	}
	steps := []step{
		{400 * time.Millisecond, "info", fmt.Sprintf("基建模式：%s", modeName)},
	}
	if plan := optString(options, "plan", ""); plan != "" {
		steps = append(steps, step{300 * time.Millisecond, "info", fmt.Sprintf("排班文件：%s", plan)})
	}
	steps = append(steps,
		step{500 * time.Millisecond, "info", "进入基建，读取当前排班…"},
		step{500 * time.Millisecond, "info", "贸易站换班完成"},
		step{500 * time.Millisecond, "info", "制造站换班完成"},
		step{400 * time.Millisecond, "info", fmt.Sprintf("无人机用途：%s", optString(options, "drone", "Money"))},
	)
	if optBool(options, "receiveCredit", true) {
		steps = append(steps, step{300 * time.Millisecond, "info", "会客室收取信用"})
	}
	if optBool(options, "clueExchange", true) {
		steps = append(steps, step{300 * time.Millisecond, "info", "进行线索交流"})
	}
	if optBool(options, "sendClue", true) {
		steps = append(steps, step{300 * time.Millisecond, "info", "赠送线索"})
	}
	steps = append(steps, step{400 * time.Millisecond, "success", "基建换班完成"})
	return steps
}

func recruitScript(options map[string]any) []step {
	maxTimes := optInt(options, "maxTimes", 4)
	steps := []step{
		{400 * time.Millisecond, "info", "打开公开招募…"},
		{500 * time.Millisecond, "info", "识别 Tags：防护 / 治疗 / 输出"},
		{500 * time.Millisecond, "info", "选择最优 Tag 组合"},
		{400 * time.Millisecond, "info", fmt.Sprintf("开始招募，最多 %d 次", maxTimes)},
	}
	if optBool(options, "expedited", false) {
		steps = append(steps, step{300 * time.Millisecond, "warn", "已启用加急许可"})
	}
	steps = append(steps,
		step{600 * time.Millisecond, "info", "第 1 次招募已开始"},
		step{600 * time.Millisecond, "info", "第 2 次招募已开始"},
		step{400 * time.Millisecond, "success", "自动公招完成"},
	)
	return steps
}

func mallScript(options map[string]any) []step {
	steps := []step{}
	if optBool(options, "visitFriends", true) {
		steps = append(steps,
			step{400 * time.Millisecond, "info", "访问好友…"},
			step{500 * time.Millisecond, "success", "好友访问完成"},
		)
	}
	if optBool(options, "shopping", true) {
		steps = append(steps,
			step{400 * time.Millisecond, "info", "打开信用交易所…"},
			step{500 * time.Millisecond, "info", fmt.Sprintf("优先购买：%s", optString(options, "highPriority", "招聘许可;加急许可"))},
			step{500 * time.Millisecond, "success", "信用商店购物完成"},
		)
	}
	if optBool(options, "creditFight", false) {
		steps = append(steps, step{500 * time.Millisecond, "info", "借助战打 OF-1 获取信用"})
	}
	if len(steps) == 0 {
		steps = append(steps, step{400 * time.Millisecond, "warn", "未启用任何信用收支选项"})
	}
	steps = append(steps, step{400 * time.Millisecond, "success", "信用收支完成"})
	return steps
}

func awardScript(options map[string]any) []step {
	steps := []step{}
	add := func(enabled bool, message string) {
		if enabled {
			steps = append(steps, step{350 * time.Millisecond, "info", message})
		}
	}
	add(optBool(options, "dailyWeekly", true), "领取每日/每周任务奖励")
	add(optBool(options, "mail", true), "领取所有邮件奖励")
	add(optBool(options, "orundum", true), "领取幸运墙合成玉")
	add(optBool(options, "mining", true), "领取限时开采许可")
	add(optBool(options, "specialAccess", true), "领取周年赠送月卡")
	add(optBool(options, "signIn", true), "领取限时签到活动奖励")
	add(optBool(options, "freeGacha", false), "进行限定池免费单抽")
	if len(steps) == 0 {
		steps = append(steps, step{350 * time.Millisecond, "warn", "未启用任何领取选项"})
	}
	steps = append(steps, step{400 * time.Millisecond, "success", "领取奖励完成"})
	return steps
}

func clientTypeName(value string) string {
	switch value {
	case "Official":
		return "官服"
	case "YoStarEN":
		return "美服"
	case "YoStarJP":
		return "日服"
	case "YoStarKR":
		return "韩服"
	case "txwy":
		return "台服"
	default:
		return value
	}
}

func optString(options map[string]any, key, fallback string) string {
	if options == nil {
		return fallback
	}
	if value, ok := options[key]; ok {
		if text, ok := value.(string); ok {
			return text
		}
	}
	return fallback
}

func optInt(options map[string]any, key string, fallback int) int {
	if options == nil {
		return fallback
	}
	switch value := options[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case float32:
		return int(value)
	default:
		return fallback
	}
}

func optBool(options map[string]any, key string, fallback bool) bool {
	if options == nil {
		return fallback
	}
	if value, ok := options[key].(bool); ok {
		return value
	}
	return fallback
}
