package maa

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"MaaGO/internal/config"
	"MaaGO/internal/stages"
)

// TaskTypeName maps the config task type to the MaaCore task name.
func TaskTypeName(taskType config.TaskType) string {
	switch taskType {
	case config.TaskStartUp:
		return "StartUp"
	case config.TaskFight:
		return "Fight"
	case config.TaskInfrast:
		return "Infrast"
	case config.TaskRecruit:
		return "Recruit"
	case config.TaskMall:
		return "Mall"
	case config.TaskAward:
		return "Award"
	default:
		return ""
	}
}

// TaskTypeFromChain maps a MaaCore taskchain value back to a config type.
func TaskTypeFromChain(chain string) (config.TaskType, bool) {
	switch chain {
	case "StartUp":
		return config.TaskStartUp, true
	case "Fight":
		return config.TaskFight, true
	case "Infrast":
		return config.TaskInfrast, true
	case "Recruit":
		return config.TaskRecruit, true
	case "Mall":
		return config.TaskMall, true
	case "Award":
		return config.TaskAward, true
	default:
		return "", false
	}
}

// BuildTaskParams converts one UI task into the MaaCore JSON schema.
func BuildTaskParams(task config.TaskConfig, cfg config.AppConfig) (string, error) {
	return BuildTaskParamsWithStage(task, cfg, "")
}

// BuildTaskParamsWithStage is BuildTaskParams with an explicit stage override.
// The runner uses it after resolving the ordered stage plan against today's
// availability.
func BuildTaskParamsWithStage(task config.TaskConfig, cfg config.AppConfig, stageOverride string) (string, error) {
	options := task.Options
	var params map[string]any

	switch task.Type {
	case config.TaskStartUp:
		clientType := optString(options, "clientType", cfg.Run.ClientType)
		if clientType == "" {
			clientType = "Official"
		}
		params = map[string]any{
			"enable":             true,
			"client_type":        clientType,
			"start_game_enabled": optBool(options, "launchClient", true),
		}
		if account := strings.TrimSpace(optString(options, "accountSwitch", "")); account != "" {
			params["account_name"] = account
		}

	case config.TaskFight:
		times := optInt(options, "times", 0)
		if times <= 0 {
			times = math.MaxInt32
		}
		// MaaCore consumes normal potions before it applies the expiring window,
		// so keep the two budgets explicit just like the official client.
		medicine := 0
		if optBool(options, "medicine", false) {
			medicine = optInt(options, "medicineCount", 1)
			if medicine < 1 {
				medicine = 1
			}
		}
		expireDays := 0
		if optBool(options, "expiringMedicine", false) {
			expireDays = optInt(options, "medicineExpireDays", 1)
			if expireDays < 1 {
				expireDays = 1
			}
			if expireDays > 7 {
				expireDays = 7
			}
		}
		stone := 0
		if optBool(options, "stone", false) {
			stone = optInt(options, "stoneCount", 1)
			if stone < 1 {
				stone = 1
			}
		}
		params = map[string]any{
			"enable":               true,
			"medicine":             medicine,
			"medicine_expire_days": expireDays,
			"stone":                stone,
			"times":                times,
			"series":               optInt(options, "series", 0),
		}
		stage := strings.TrimSpace(stageOverride)
		if stage == "" {
			stage = strings.TrimSpace(optString(options, "stage", ""))
		}
		if stage == "" {
			for _, candidate := range stages.Ordered(options) {
				if candidate = strings.TrimSpace(candidate); candidate != "" {
					stage = candidate
					break
				}
			}
		}
		if stage != "" {
			params["stage"] = stage
		}
		if clientType := cfg.Run.ClientType; clientType != "" {
			params["client_type"] = clientType
			if server := serverForClient(clientType); server != "" {
				params["server"] = server
			}
		}

	case config.TaskInfrast:
		mode := 0
		filename := strings.TrimSpace(optString(options, "plan", ""))
		switch optString(options, "mode", "default") {
		case "rotation":
			mode = 20000
		case "custom":
			if filename != "" {
				mode = 10000
			}
		}
		facilities := stringSlice(options["facilities"])
		if len(facilities) == 0 {
			facilities = []string{"Mfg", "Trade", "Power", "Control", "Reception", "Office", "Dorm"}
		}
		threshold := float64(optInt(options, "threshold", 30)) / 100
		params = map[string]any{
			"enable":                    true,
			"mode":                      mode,
			"facility":                  facilities,
			"drones":                    optString(options, "drone", "Money"),
			"threshold":                 math.Round(threshold*100) / 100,
			"reception_message_board":   optBool(options, "receiveCredit", true),
			"reception_clue_exchange":   optBool(options, "clueExchange", true),
			"reception_send_clue":       optBool(options, "sendClue", true),
			"continue_training":         false,
			"dorm_trust_enabled":        false,
			"dorm_notstationed_enabled": false,
			"replenish":                 false,
		}
		if mode == 10000 {
			params["filename"] = filename
			params["plan_index"] = 0
		}

	case config.TaskRecruit:
		confirm := make([]int, 0, 3)
		selectList := make([]int, 0, 3)
		if optBool(options, "confirm3", true) {
			confirm = append(confirm, 3)
		}
		if optBool(options, "confirm4", true) {
			selectList = append(selectList, 4)
			confirm = append(confirm, 4)
		}
		if optBool(options, "confirm5", false) {
			selectList = append(selectList, 5)
			confirm = append(confirm, 5)
		}
		extraMode := 0
		switch optString(options, "strategy", "default") {
		case "rare":
			extraMode = 1
		case "highStar":
			extraMode = 2
		}
		params = map[string]any{
			"enable":          true,
			"refresh":         optBool(options, "autoRefresh", true),
			"select":          selectList,
			"confirm":         confirm,
			"times":           optInt(options, "maxTimes", 4),
			"set_time":        true,
			"expedite":        optBool(options, "expedited", false),
			"extra_tags_mode": extraMode,
		}
		if tags := splitList(optString(options, "preserveTags", "")); len(tags) > 0 {
			params["preserve_tags"] = tags
		}
		if server := serverForClient(cfg.Run.ClientType); server != "" {
			params["server"] = server
		}

	case config.TaskMall:
		params = map[string]any{
			"enable":                        true,
			"visit_friends":                 optBool(options, "visitFriends", true),
			"shopping":                      optBool(options, "shopping", true),
			"buy_first":                     splitList(optString(options, "highPriority", "")),
			"blacklist":                     splitList(optString(options, "blacklist", "")),
			"force_shopping_if_credit_full": false,
			"only_buy_discount":             optBool(options, "onlyDiscount", false),
			"reserve_max_credit":            optBool(options, "reserveCredit", true),
			"credit_fight":                  optBool(options, "creditFight", false),
			"formation_index":               0,
		}

	case config.TaskAward:
		params = map[string]any{
			"enable":        true,
			"award":         optBool(options, "dailyWeekly", true),
			"mail":          optBool(options, "mail", true),
			"recruit":       optBool(options, "freeGacha", false),
			"orundum":       optBool(options, "orundum", true),
			"mining":        optBool(options, "mining", true),
			"specialaccess": optBool(options, "specialAccess", true),
			"signinevent":   optBool(options, "signIn", true),
		}

	default:
		return "", fmt.Errorf("unsupported task type %q", task.Type)
	}

	data, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("encode %s task: %w", task.Type, err)
	}
	return string(data), nil
}

// ConnectionConfigName maps the UI emulator selection to resource/config.json.
func ConnectionConfigName(emulator string) string {
	switch emulator {
	case "mumu":
		return "MuMuEmulator12"
	case "ld":
		return "LDPlayer"
	case "bluestacks":
		return "BlueStacks"
	case "nox":
		return "Nox"
	case "memu":
		return "XYAZ"
	default:
		return "General"
	}
}

func serverForClient(clientType string) string {
	switch clientType {
	case "Official", "Bilibili":
		return "CN"
	case "YoStarEN":
		return "US"
	case "YoStarJP":
		return "JP"
	case "YoStarKR":
		return "KR"
	default:
		return ""
	}
}

func splitList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ';' || r == '；' || r == '\n' || r == ','
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if text := strings.TrimSpace(field); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func stringSlice(value any) []string {
	switch items := value.(type) {
	case []string:
		return append([]string(nil), items...)
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if text, ok := item.(string); ok && text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func optString(options map[string]any, key, fallback string) string {
	if options == nil {
		return fallback
	}
	if value, ok := options[key].(string); ok {
		return value
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
