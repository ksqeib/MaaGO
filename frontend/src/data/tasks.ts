import type { LocalizedText, TaskConfig, TaskType } from "@/types/maa";

export type TaskIcon = "rocket" | "swords" | "factory" | "users" | "shopping" | "gift";

export type FieldType = "switch" | "text" | "number" | "select" | "multiselect";

export interface FieldOption {
  value: string;
  label: LocalizedText;
}

export interface FieldDef {
  key: string;
  type: FieldType;
  label: LocalizedText;
  hint?: LocalizedText;
  placeholder?: LocalizedText;
  min?: number;
  max?: number;
  step?: number;
  options?: FieldOption[];
  defaultValue?: unknown;
}

export interface TaskMeta {
  type: TaskType;
  icon: TaskIcon;
  label: LocalizedText;
  description: LocalizedText;
  fields: FieldDef[];
}

const clientOptions: FieldOption[] = [
  { value: "Official", label: { zh: "官服", en: "Official" } },
  { value: "YoStarEN", label: { zh: "美服", en: "YoStarEN" } },
  { value: "YoStarJP", label: { zh: "日服", en: "YoStarJP" } },
  { value: "YoStarKR", label: { zh: "韩服", en: "YoStarKR" } },
  { value: "txwy", label: { zh: "台服", en: "txwy" } },
];

export const TASK_META: TaskMeta[] = [
  {
    type: "startup",
    icon: "rocket",
    label: { zh: "开始唤醒", en: "Start up" },
    description: {
      zh: "连接模拟器、启动客户端并进入游戏主界面。",
      en: "Connect to the emulator, launch the client and reach the main screen.",
    },
    fields: [
      {
        key: "launchClient",
        type: "switch",
        label: { zh: "自动启动客户端", en: "Launch client automatically" },
        defaultValue: true,
      },
      {
        key: "clientType",
        type: "select",
        label: { zh: "客户端类型", en: "Client type" },
        options: clientOptions,
        defaultValue: "Official",
      },
      {
        key: "accountSwitch",
        type: "text",
        label: { zh: "账号切换", en: "Account switch" },
        placeholder: { zh: "留空保持当前账号", en: "Leave empty to keep the current account" },
        defaultValue: "",
      },
    ],
  },
  {
    type: "fight",
    icon: "swords",
    label: { zh: "理智作战", en: "Sanity fight" },
    description: {
      zh: "按指定关卡和次数消耗理智，支持自动补充药剂。",
      en: "Spend sanity on a stage, with optional medicine refills.",
    },
    fields: [
      {
        key: "stage",
        type: "text",
        label: { zh: "关卡", en: "Stage" },
        placeholder: { zh: "留空使用当前/上次关卡", en: "Leave empty for the current/last stage" },
        defaultValue: "",
      },
      {
        key: "times",
        type: "number",
        label: { zh: "作战次数", en: "Battle count" },
        hint: { zh: "0 表示直到理智耗尽", en: "0 means until sanity runs out" },
        min: 0,
        max: 999,
        defaultValue: 0,
      },
      {
        key: "medicine",
        type: "switch",
        label: { zh: "使用理智药", en: "Use sanity potions" },
        defaultValue: true,
      },
      {
        key: "stone",
        type: "switch",
        label: { zh: "使用源石", en: "Use Originite Prime" },
        defaultValue: false,
      },
      {
        key: "series",
        type: "select",
        label: { zh: "代理倍率", en: "Auto-repeat multiplier" },
        options: [
          { value: "1", label: { zh: "1 倍", en: "1x" } },
          { value: "2", label: { zh: "2 倍", en: "2x" } },
          { value: "3", label: { zh: "3 倍", en: "3x" } },
          { value: "4", label: { zh: "4 倍", en: "4x" } },
          { value: "6", label: { zh: "6 倍", en: "6x" } },
        ],
        defaultValue: "1",
      },
    ],
  },
  {
    type: "infrast",
    icon: "factory",
    label: { zh: "基建换班", en: "Infrastructure" },
    description: {
      zh: "自动收取线索、无人机与信用，并按策略换班。",
      en: "Collect clues, drones and credits, and rotate operators.",
    },
    fields: [
      {
        key: "mode",
        type: "select",
        label: { zh: "基建模式", en: "Mode" },
        options: [
          { value: "default", label: { zh: "常规模式", en: "Default" } },
          { value: "rotation", label: { zh: "队列轮换", en: "Rotation" } },
          { value: "custom", label: { zh: "自定义排班", en: "Custom plan" } },
        ],
        defaultValue: "default",
      },
      {
        key: "plan",
        type: "text",
        label: { zh: "排班文件", en: "Plan file" },
        placeholder: { zh: "留空使用内置配置", en: "Leave empty for the built-in plan" },
        defaultValue: "",
      },
      {
        key: "drone",
        type: "select",
        label: { zh: "无人机用途", en: "Drone usage" },
        options: [
          { value: "Money", label: { zh: "贸易站", en: "Trading post" } },
          { value: "SyntheticJade", label: { zh: "合成玉", en: "Synthetic jade" } },
          { value: "CombatRecord", label: { zh: "经验", en: "Combat records" } },
          { value: "PureGold", label: { zh: "赤金", en: "Pure gold" } },
          { value: "OriginStone", label: { zh: "源石碎片", en: "Originium shards" } },
        ],
        defaultValue: "Money",
      },
      {
        key: "threshold",
        type: "number",
        label: { zh: "心情阈值", en: "Mood threshold" },
        min: 0,
        max: 100,
        defaultValue: 30,
      },
      {
        key: "facilities",
        type: "multiselect",
        label: { zh: "换班设施", en: "Facilities" },
        options: [
          { value: "Mfg", label: { zh: "制造站", en: "Factory" } },
          { value: "Trade", label: { zh: "贸易站", en: "Trading post" } },
          { value: "Power", label: { zh: "发电站", en: "Power plant" } },
          { value: "Control", label: { zh: "控制中枢", en: "Control center" } },
          { value: "Reception", label: { zh: "会客室", en: "Reception room" } },
          { value: "Office", label: { zh: "办公室", en: "Office" } },
          { value: "Dorm", label: { zh: "宿舍", en: "Dormitory" } },
        ],
        defaultValue: ["Mfg", "Trade", "Power", "Control", "Reception", "Office", "Dorm"],
      },
      {
        key: "receiveCredit",
        type: "switch",
        label: { zh: "会客室收取信用", en: "Collect reception credits" },
        defaultValue: true,
      },
      {
        key: "clueExchange",
        type: "switch",
        label: { zh: "进行线索交流", en: "Exchange clues" },
        defaultValue: true,
      },
      {
        key: "sendClue",
        type: "switch",
        label: { zh: "赠送线索", en: "Send clues" },
        defaultValue: true,
      },
    ],
  },
  {
    type: "recruit",
    icon: "users",
    label: { zh: "自动公招", en: "Recruitment" },
    description: {
      zh: "识别公招 Tags，按策略选择组合并自动招募。",
      en: "Recognise recruitment tags, pick combinations and recruit.",
    },
    fields: [
      {
        key: "maxTimes",
        type: "number",
        label: { zh: "最大招募次数", en: "Max recruitments" },
        min: 0,
        max: 10,
        defaultValue: 4,
      },
      {
        key: "expedited",
        type: "switch",
        label: { zh: "自动使用加急许可", en: "Use expedited plans" },
        defaultValue: false,
      },
      {
        key: "confirm3",
        type: "switch",
        label: { zh: "自动确认 3 星", en: "Confirm 3 stars" },
        defaultValue: true,
      },
      {
        key: "confirm4",
        type: "switch",
        label: { zh: "自动确认 4 星", en: "Confirm 4 stars" },
        defaultValue: false,
      },
      {
        key: "confirm5",
        type: "switch",
        label: { zh: "自动确认 5 星", en: "Confirm 5 stars" },
        defaultValue: false,
      },
      {
        key: "autoRefresh",
        type: "switch",
        label: { zh: "自动刷新 3 星 Tags", en: "Refresh 3-star tags" },
        defaultValue: true,
      },
      {
        key: "strategy",
        type: "select",
        label: { zh: "多选 Tag 策略", en: "Tag strategy" },
        options: [
          { value: "default", label: { zh: "默认", en: "Default" } },
          { value: "highStar", label: { zh: "优先高星", en: "Prefer high stars" } },
          { value: "rare", label: { zh: "优先稀有", en: "Prefer rare" } },
        ],
        defaultValue: "default",
      },
      {
        key: "preserveTags",
        type: "text",
        label: { zh: "保留指定词条", en: "Preserve tags" },
        placeholder: { zh: "多个词条用分号分隔", en: "Separate tags with semicolons" },
        defaultValue: "",
      },
    ],
  },
  {
    type: "mall",
    icon: "shopping",
    label: { zh: "信用收支", en: "Credit store" },
    description: {
      zh: "访问好友、领取信用并自动购买信用商店商品。",
      en: "Visit friends and buy from the credit store.",
    },
    fields: [
      {
        key: "visitFriends",
        type: "switch",
        label: { zh: "访问好友", en: "Visit friends" },
        defaultValue: true,
      },
      {
        key: "onceADay",
        type: "switch",
        label: { zh: "一日只执行一次", en: "Only once a day" },
        defaultValue: true,
      },
      {
        key: "creditFight",
        type: "switch",
        label: { zh: "借助战打 OF-1 赚信用", en: "OF-1 support battle" },
        defaultValue: false,
      },
      {
        key: "shopping",
        type: "switch",
        label: { zh: "自动购物", en: "Auto shopping" },
        defaultValue: true,
      },
      {
        key: "highPriority",
        type: "text",
        label: { zh: "优先购买", en: "High priority" },
        placeholder: { zh: "子串匹配，分号分隔", en: "Substring match, semicolon separated" },
        defaultValue: "招聘许可;加急许可",
      },
      {
        key: "blacklist",
        type: "text",
        label: { zh: "黑名单", en: "Blacklist" },
        placeholder: { zh: "子串匹配，分号分隔", en: "Substring match, semicolon separated" },
        defaultValue: "碳;家具",
      },
      {
        key: "onlyDiscount",
        type: "switch",
        label: { zh: "只购买打折商品", en: "Only discounted items" },
        defaultValue: false,
      },
      {
        key: "reserveCredit",
        type: "number",
        label: { zh: "信用低于此值停止", en: "Stop below credits" },
        min: 0,
        max: 1000,
        defaultValue: 300,
      },
    ],
  },
  {
    type: "award",
    icon: "gift",
    label: { zh: "领取奖励", en: "Collect rewards" },
    description: {
      zh: "领取每日/每周、邮件、活动与签到奖励。",
      en: "Collect daily, weekly, mail, event and sign-in rewards.",
    },
    fields: [
      {
        key: "dailyWeekly",
        type: "switch",
        label: { zh: "每日/每周任务奖励", en: "Daily and weekly missions" },
        defaultValue: true,
      },
      {
        key: "mail",
        type: "switch",
        label: { zh: "邮件奖励", en: "Mail rewards" },
        defaultValue: true,
      },
      {
        key: "orundum",
        type: "switch",
        label: { zh: "幸运墙合成玉", en: "Lucky wall orundum" },
        defaultValue: true,
      },
      {
        key: "mining",
        type: "switch",
        label: { zh: "限时开采许可", en: "Limited mining permit" },
        defaultValue: true,
      },
      {
        key: "specialAccess",
        type: "switch",
        label: { zh: "周年赠送月卡", en: "Anniversary monthly card" },
        defaultValue: true,
      },
      {
        key: "signIn",
        type: "switch",
        label: { zh: "限时签到活动", en: "Limited sign-in event" },
        defaultValue: true,
      },
      {
        key: "freeGacha",
        type: "switch",
        label: { zh: "限定池免费单抽", en: "Free daily pull" },
        defaultValue: false,
      },
    ],
  },
];

export const TASK_META_BY_TYPE = Object.fromEntries(
  TASK_META.map((meta) => [meta.type, meta]),
) as Record<TaskType, TaskMeta>;

export function buildDefaultOptions(meta: TaskMeta): Record<string, unknown> {
  const options: Record<string, unknown> = {};
  for (const field of meta.fields) {
    if (field.defaultValue !== undefined) {
      options[field.key] = Array.isArray(field.defaultValue)
        ? [...field.defaultValue]
        : field.defaultValue;
    } else if (field.type === "switch") {
      options[field.key] = false;
    } else if (field.type === "number") {
      options[field.key] = 0;
    } else if (field.type === "multiselect") {
      options[field.key] = [];
    } else {
      options[field.key] = "";
    }
  }
  return options;
}

export function createTask(type: TaskType): TaskConfig {
  const meta = TASK_META_BY_TYPE[type];
  return {
    id: `task-${type}-${Date.now()}`,
    type,
    name: "",
    enabled: true,
    options: buildDefaultOptions(meta),
  };
}

export function defaultTasks(): TaskConfig[] {
  return TASK_META.map((meta) => {
    const task = createTask(meta.type);
    task.id = `task-${meta.type}`;
    task.enabled = true;
    return task;
  });
}
