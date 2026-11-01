import { TASK_META_BY_TYPE } from "@/data/tasks";
import type { TaskConfig } from "@/types/maa";
import { pickText } from "@/utils/text";

function text(language: string | undefined, zh: string, en: string) {
  return String(language || "").startsWith("zh") ? zh : en;
}

export function taskSummary(task: TaskConfig, language?: string): string {
  const options = task.options ?? {};
  switch (task.type) {
    case "startup": {
      const client = String(options.clientType ?? "Official");
      const map: Record<string, [string, string]> = {
        Official: ["官服", "Official"],
        YoStarEN: ["美服", "YoStarEN"],
        YoStarJP: ["日服", "YoStarJP"],
        YoStarKR: ["韩服", "YoStarKR"],
        txwy: ["台服", "txwy"],
      };
      const label = map[client] ?? [client, client];
      return text(language, `客户端：${label[0]}`, `Client: ${label[1]}`);
    }
    case "fight": {
      const stage = String(options.stage ?? "");
      const times = Number(options.times ?? 0);
      const stageText = stage || text(language, "当前/上次", "Current/last");
      return times > 0
        ? text(language, `${stageText} · ${times} 次`, `${stageText} · ${times}x`)
        : text(language, `${stageText} · 直到理智耗尽`, `${stageText} · until sanity ends`);
    }
    case "infrast": {
      const mode = String(options.mode ?? "default");
      const map: Record<string, [string, string]> = {
        default: ["常规模式", "Default"],
        rotation: ["队列轮换", "Rotation"],
        custom: ["自定义", "Custom"],
      };
      const label = map[mode] ?? [mode, mode];
      return text(language, `基建模式：${label[0]}`, `Mode: ${label[1]}`);
    }
    case "recruit": {
      const max = Number(options.maxTimes ?? 0);
      return text(language, `最多招募 ${max} 次`, `Up to ${max} recruitments`);
    }
    case "mall":
      return text(
        language,
        options.shopping !== false ? "访问好友 + 自动购物" : "访问好友",
        options.shopping !== false ? "Friends + shopping" : "Friends",
      );
    case "award": {
      const count = Object.values(options).filter((value) => value === true).length;
      return text(language, `已选 ${count} 项奖励`, `${count} reward groups`);
    }
    default:
      return "";
  }
}

export function taskDisplayName(task: TaskConfig, language?: string): string {
  if (task.name) return task.name;
  return pickText(TASK_META_BY_TYPE[task.type].label, language);
}
