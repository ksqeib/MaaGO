import { useEffect, useRef, useState, type KeyboardEvent, type MouseEvent } from "react";
import { useTranslation } from "react-i18next";
import { FaCompress, FaExpand, FaMousePointer, FaPlug } from "react-icons/fa";

import { Switch } from "@/components/Switch";
import { cn } from "@/utils/cn";
import * as api from "bindings/MaaGO/internal/app/service";

interface Preview {
  connected: boolean;
  host: string;
  username: string;
  width: number;
  height: number;
  imageWidth: number;
  imageHeight: number;
  image: string;
  error?: string;
}

const keyMap: Record<string, string> = {
  Enter: "enter",
  Escape: "esc",
  Tab: "tab",
  Backspace: "backspace",
  Delete: "delete",
  ArrowUp: "up",
  ArrowDown: "down",
  ArrowLeft: "left",
  ArrowRight: "right",
};

export const RDPPreviewCard = () => {
  const { t } = useTranslation();
  const [preview, setPreview] = useState<Preview | null>(null);
  const [open, setOpen] = useState(false);
  const [operate, setOperate] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [text, setText] = useState("");
  const modalRef = useRef<HTMLDivElement>(null);

  const maxWidth = open ? 1280 : 640;

  useEffect(() => {
    let alive = true;
    let timer = 0;
    const load = async () => {
      try {
        const next = (await api.GetRDPPreview(maxWidth)) as unknown as Preview;
        if (alive) setPreview(next);
      } catch {
        // Keep the last frame; the next poll may recover.
      }
    };
    void load();
    // The enlarged view is used for operating the remote, so refresh it faster.
    timer = window.setInterval(load, maxWidth >= 1280 ? 250 : 500);
    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, [maxWidth]);

  useEffect(() => {
    if (operate) modalRef.current?.focus();
  }, [operate]);

  const connect = async () => {
    setConnecting(true);
    try {
      const next = (await api.ConnectRDP()) as unknown as Preview;
      setPreview(next);
    } catch {
      // The preview poll reports the error text.
    } finally {
      setConnecting(false);
    }
  };

  const send = (action: string, x = 0, y = 0, body = "", key = "") => {
    void api.SendRDPInput(action, x, y, body, key).catch(() => undefined);
  };

  const handleImageClick = (event: MouseEvent<HTMLImageElement>) => {
    if (!operate || !preview) return;
    const rect = event.currentTarget.getBoundingClientRect();
    const x = Math.round(((event.clientX - rect.left) / rect.width) * preview.width);
    const y = Math.round(((event.clientY - rect.top) / rect.height) * preview.height);
    send("click", x, y);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!operate) return;
    const key = keyMap[event.key];
    if (key) {
      event.preventDefault();
      send("key", 0, 0, "", key);
      return;
    }
    if (event.key.length === 1 && !event.ctrlKey && !event.altKey && !event.metaKey) {
      event.preventDefault();
      send("type", 0, 0, event.key);
    }
  };

  const sendText = () => {
    if (!operate || !text) return;
    send("type", 0, 0, text);
    setText("");
  };

  const closeModal = () => {
    setOpen(false);
    setOperate(false);
  };

  return (
    <>
      <section className="launcher-glass rounded-3xl p-5">
        <div className="flex items-center justify-between">
          <h2 className="flex items-center gap-2 text-sm font-bold">
            <FaPlug className="text-brand-500" />
            {t("dashboard.rdp_preview")}
          </h2>
          <span
            className={cn(
              "rounded-full px-2 py-0.5 text-[10px] font-semibold",
              preview?.connected
                ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
                : "bg-surface-secondary text-muted",
            )}
          >
            {preview?.connected
              ? t("dashboard.rdp_preview_connected")
              : t("dashboard.rdp_preview_disconnected")}
          </span>
        </div>

        <button
          className="mt-3 block w-full overflow-hidden rounded-2xl border border-border bg-black/40 text-left"
          onClick={() => {
            setOpen(true);
            setOperate(false);
          }}
          type="button"
        >
          {preview?.connected && preview.image ? (
            <img
              alt="RDP preview"
              className="aspect-video w-full object-cover"
              src={preview.image}
            />
          ) : (
            <div className="flex aspect-video w-full flex-col items-center justify-center gap-1 text-[11px] text-muted">
              <FaMousePointer />
              <span>{preview?.error || t("dashboard.rdp_preview_empty")}</span>
            </div>
          )}
        </button>

        <div className="mt-2 flex gap-2">
          <button
            className="flex flex-1 items-center justify-center gap-1.5 rounded-full bg-brand-500 px-3 py-1.5 text-[11px] font-semibold text-white hover:bg-brand-600 disabled:opacity-60"
            disabled={!preview?.connected}
            onClick={() => {
              setOpen(true);
              setOperate(false);
            }}
            type="button"
          >
            <FaExpand size={9} />
            {t("dashboard.rdp_preview_enlarge")}
          </button>
          {!preview?.connected && (
            <button
              className="flex flex-1 items-center justify-center gap-1.5 rounded-full bg-surface-secondary px-3 py-1.5 text-[11px] font-medium text-muted hover:text-foreground disabled:opacity-60"
              disabled={connecting}
              onClick={() => void connect()}
              type="button"
            >
              <FaPlug size={9} />
              {connecting ? t("settings.connection.testing") : t("dashboard.rdp_preview_connect")}
            </button>
          )}
        </div>
        <p className="mt-2 text-[10px] leading-relaxed text-muted">
          {t("dashboard.rdp_preview_hint")}
        </p>
      </section>

      {open && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4"
          onClick={closeModal}
        >
          <div
            className="launcher-glass max-h-[95vh] w-full max-w-[1280px] overflow-auto rounded-3xl p-4"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h2 className="flex items-center gap-2 text-sm font-bold">
                <FaPlug className="text-brand-500" />
                {t("dashboard.rdp_preview_title")}
                <span className="font-mono text-[10px] font-normal text-muted">
                  {preview?.username}@{preview?.host}
                </span>
              </h2>
              <div className="flex items-center gap-3">
                <label className="flex items-center gap-2 text-[11px] text-muted">
                  {t("dashboard.rdp_operate")}
                  <Switch checked={operate} onChange={setOperate} />
                </label>
                <button
                  className="rounded-full bg-surface-secondary px-3 py-1 text-[11px] font-medium text-muted hover:text-foreground"
                  onClick={closeModal}
                  type="button"
                >
                  <FaCompress size={9} />
                </button>
              </div>
            </div>

            <div
              className="relative mt-3 outline-none"
              onKeyDown={handleKeyDown}
              ref={modalRef}
              tabIndex={0}
            >
              {preview?.connected && preview.image ? (
                <img
                  alt="RDP operation"
                  className={cn(
                    "w-full rounded-2xl",
                    operate ? "cursor-crosshair" : "cursor-default",
                  )}
                  onClick={handleImageClick}
                  src={preview.image}
                />
              ) : (
                <div className="flex aspect-video w-full items-center justify-center rounded-2xl bg-black/40 text-xs text-muted">
                  {preview?.error || t("dashboard.rdp_preview_empty")}
                </div>
              )}
              {!operate && (
                <div className="pointer-events-none absolute inset-0 flex items-center justify-center rounded-2xl bg-black/25">
                  <span className="rounded-full bg-black/60 px-3 py-1.5 text-[11px] font-medium text-white">
                    {t("dashboard.rdp_preview_only")}
                  </span>
                </div>
              )}
            </div>

            {operate && (
              <div className="mt-3 flex flex-wrap items-center gap-2">
                <input
                  className="h-9 min-w-[12rem] flex-1 rounded-xl border border-border bg-surface-secondary/60 px-3 text-sm outline-none focus:border-brand-500/60"
                  onChange={(event) => setText(event.target.value)}
                  onKeyDown={(event) => {
                    event.stopPropagation();
                    if (event.key === "Enter") {
                      event.preventDefault();
                      sendText();
                    }
                  }}
                  placeholder={t("dashboard.rdp_text_placeholder")}
                  value={text}
                />
                <button
                  className="rounded-full bg-brand-500 px-3 py-1.5 text-[11px] font-semibold text-white hover:bg-brand-600"
                  onClick={sendText}
                  type="button"
                >
                  {t("dashboard.rdp_send")}
                </button>
                <button
                  className="rounded-full bg-surface-secondary px-3 py-1.5 text-[11px] font-medium text-muted hover:text-foreground"
                  onClick={() => send("key", 0, 0, "", "enter")}
                  type="button"
                >
                  Enter
                </button>
                <button
                  className="rounded-full bg-surface-secondary px-3 py-1.5 text-[11px] font-medium text-muted hover:text-foreground"
                  onClick={() => send("key", 0, 0, "", "esc")}
                  type="button"
                >
                  Esc
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
};
