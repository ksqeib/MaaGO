import type { LocalizedText } from "@/types/maa";

export function pickText(text: LocalizedText, language?: string): string {
  return String(language || "").startsWith("zh") ? text.zh : text.en;
}
