// طرفية الويب — xterm.js + WebSocket للوصول CMD (SUPER_ADMIN فقط)
"use client";

import React, { useEffect, useRef, useState, useCallback } from "react";
import "@xterm/xterm/css/xterm.css";
import { getAccessToken, getUserRole } from "@/lib/auth";
import { getSwitchUrl } from "@/lib/api";

export default function TerminalPage() {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<any>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const fitAddonRef = useRef<any>(null);
  const pingRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [shellInfo, setShellInfo] = useState<{ shell: string; os: string } | null>(null);

  // التحقق من الدور
  const role = getUserRole() || "VIEWER";
  const isSuperAdmin = role === "SUPER_ADMIN";

  // تنظيف الموارد
  const cleanup = useCallback(() => {
    if (pingRef.current) {
      clearInterval(pingRef.current);
      pingRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setConnected(false);
  }, []);

  // إعداد xterm.js و WebSocket
  useEffect(() => {
    if (!isSuperAdmin || !terminalRef.current) return;

    let disposed = false;

    const initTerminal = async () => {
      // استيراد ديناميكي لأن xterm يحتاج كائن window
      const { Terminal } = await import("@xterm/xterm");
      const { FitAddon } = await import("@xterm/addon-fit");

      if (disposed) return;

      const terminal = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: "'Cascadia Code', 'Fira Code', 'Consolas', monospace",
        theme: {
          background: "#0d1117",
          foreground: "#c9d1d9",
          cursor: "#58a6ff",
          selectionBackground: "#264f78",
          black: "#484f58",
          red: "#ff7b72",
          green: "#3fb950",
          yellow: "#d29922",
          blue: "#58a6ff",
          magenta: "#bc8cff",
          cyan: "#39c5cf",
          white: "#b1bac4",
          brightBlack: "#6e7681",
          brightRed: "#ffa198",
          brightGreen: "#56d364",
          brightYellow: "#e3b341",
          brightBlue: "#79c0ff",
          brightMagenta: "#d2a8ff",
          brightCyan: "#56d4dd",
          brightWhite: "#f0f6fc",
        },
        allowProposedApi: true,
      });

      const fitAddon = new FitAddon();
      terminal.loadAddon(fitAddon);
      terminal.open(terminalRef.current!);

      // تأخير fit قليلاً حتى يكتمل العرض
      setTimeout(() => {
        if (!disposed) fitAddon.fit();
      }, 100);

      xtermRef.current = terminal;
      fitAddonRef.current = fitAddon;

      // الحصول على الرمز وعنوان السويتش
      const token = getAccessToken();
      const switchUrl = getSwitchUrl();

      if (!token) {
        terminal.writeln("\x1b[31mخطأ: لم يتم العثور على رمز المصادقة. سجّل الدخول أولاً.\x1b[0m");
        setError("لم يتم العثور على رمز المصادقة");
        return;
      }

      // تحويل عنوان HTTP إلى WebSocket
      const wsUrl = switchUrl
        .replace(/^http/, "ws")
        .replace(/\/$/, "");

      terminal.writeln("\x1b[33mجارٍ الاتصال بالطرفية البعيدة...\x1b[0m");

      // إنشاء اتصال WebSocket
      const ws = new WebSocket(`${wsUrl}/admin/v1/terminal?token=${encodeURIComponent(token)}`);
      wsRef.current = ws;

      ws.onopen = () => {
        if (disposed) return;
        terminal.writeln("\x1b[32mتم الاتصال بالخادم\x1b[0m");
        terminal.writeln("");
      };

      ws.onmessage = (event) => {
        if (disposed) return;
        try {
          const msg = JSON.parse(event.data);
          switch (msg.type) {
            case "connected":
              setConnected(true);
              setShellInfo({ shell: msg.shell, os: msg.os });
              terminal.writeln(`\x1b[36m═══════════════════════════════════════\x1b[0m`);
              terminal.writeln(`\x1b[36m  طرفية Atheer البعيدة\x1b[0m`);
              terminal.writeln(`\x1b[36m  الصدفة: ${msg.shell} | النظام: ${msg.os}\x1b[0m`);
              terminal.writeln(`\x1b[36m═══════════════════════════════════════\x1b[0m`);
              terminal.writeln("");
              break;
            case "output":
              terminal.write(msg.data);
              break;
            case "error":
              terminal.write(`\x1b[31m${msg.data}\x1b[0m`);
              break;
            case "exit":
              terminal.writeln("");
              terminal.writeln(`\x1b[33mانتهت العملية برمز: ${msg.exitCode}\x1b[0m`);
              setConnected(false);
              break;
            case "pong":
              break;
          }
        } catch {
          // رسالة غير JSON — تجاهل
        }
      };

      ws.onerror = () => {
        if (disposed) return;
        terminal.writeln("\x1b[31mفشل الاتصال بالخادم\x1b[0m");
        setError("فشل الاتصال بالخادم");
        setConnected(false);
      };

      ws.onclose = (event) => {
        if (disposed) return;
        terminal.writeln("");
        terminal.writeln(`\x1b[33mانقطع الاتصال (رمز: ${event.code})\x1b[0m`);
        setConnected(false);
      };

      // إرسال الإدخال من الطرفية إلى WebSocket
      terminal.onData((data: string) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "input", data }));
        }
      });

      // إرسال أبعاد الطرفية عند تغيير الحجم
      const handleResize = () => {
        if (disposed) return;
        if (fitAddon) {
          fitAddon.fit();
          if (ws.readyState === WebSocket.OPEN && terminal.cols && terminal.rows) {
            ws.send(JSON.stringify({
              type: "resize",
              cols: terminal.cols,
              rows: terminal.rows,
            }));
          }
        }
      };

      window.addEventListener("resize", handleResize);

      // ping دوري لإبقاء الاتصال
      pingRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "ping" }));
        }
      }, 30000);
    };

    initTerminal();

    return () => {
      disposed = true;
      cleanup();
      if (xtermRef.current) {
        xtermRef.current.dispose();
        xtermRef.current = null;
      }
    };
  }, [isSuperAdmin, cleanup]);

  // إعادة الاتصال
  const handleReconnect = () => {
    setError(null);
    cleanup();
    if (xtermRef.current) {
      xtermRef.current.dispose();
      xtermRef.current = null;
    }
    // إعادة تحميل الصفحة لإعادة تهيئة xterm
    window.location.reload();
  };

  // غير مصرح
  if (!isSuperAdmin) {
    return (
      <div className="flex h-[calc(100vh-8rem)] items-center justify-center">
        <div className="text-center space-y-4">
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-destructive/10">
            <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-destructive">
              <rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          </div>
          <h2 className="text-xl font-bold text-foreground">غير مصرح</h2>
          <p className="text-muted-foreground">فقط المدير الأعلى (SUPER_ADMIN) يمكنه استخدام الطرفية</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* الشريط العلوي */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold text-foreground">الطرفية</h1>
          {shellInfo && (
            <span className="rounded-full bg-muted px-3 py-1 text-xs text-muted-foreground" dir="ltr">
              {shellInfo.shell} • {shellInfo.os}
            </span>
          )}
        </div>
        <div className="flex items-center gap-3">
          {/* مؤشر الحالة */}
          <div className="flex items-center gap-2">
            <span className={`h-2.5 w-2.5 rounded-full ${connected ? "bg-green-500 shadow-sm shadow-green-500/50" : "bg-yellow-500"}`} />
            <span className="text-sm text-muted-foreground">
              {connected ? "متصل" : "غير متصل"}
            </span>
          </div>
          {/* زر إعادة الاتصال */}
          {!connected && (
            <button
              onClick={handleReconnect}
              className="rounded-lg bg-blue-600 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-700"
            >
              إعادة الاتصال
            </button>
          )}
        </div>
      </div>

      {/* رسالة الخطأ */}
      {error && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      {/* حاوية الطرفية */}
      <div
        className="overflow-hidden rounded-lg border border-border/50 bg-[#0d1117]"
        style={{ height: "calc(100vh - 14rem)" }}
      >
        <div ref={terminalRef} className="h-full w-full" style={{ padding: "8px" }} />
      </div>
    </div>
  );
}
