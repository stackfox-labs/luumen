import { useState } from "react"
import type { ReactNode } from "react"

import type { OutputLine } from "./types"

export function lineClass(kind: OutputLine["kind"]) {
  switch (kind) {
    case "success":
      return "text-emerald-400"
    case "accent":
      return "text-rose-400"
    case "muted":
      return "text-white/35"
    case "plain":
      return "text-white/75"
    case "blank":
      return "block"
  }
}

export function TerminalWindow({
  children,
  title,
  className = "",
}: {
  children: ReactNode
  title?: string
  className?: string
}) {
  return (
    <div
      className={`bg-[#0d0d0d] rounded-xl border border-white/[0.08] overflow-hidden ${className}`}
      style={{ boxShadow: "0 20px 50px rgba(0,0,0,0.45)" }}
    >
      {title && (
        <div className="px-5 py-3 border-b border-white/[0.06]">
          <span className="font-mono text-[10px] text-white/20 tracking-wide">{title}</span>
        </div>
      )}
      {children}
    </div>
  )
}

export function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)

  return (
    <button
      onClick={async () => {
        await navigator.clipboard.writeText(text)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
      }}
      className="text-[11px] font-mono text-white/30 hover:text-white/65 transition-colors border border-white/[0.09] hover:border-white/20 px-2.5 py-1 rounded cursor-pointer"
    >
      {copied ? "copied!" : "copy"}
    </button>
  )
}

export function Pill({ children, dark = false }: { children: ReactNode; dark?: boolean }) {
  if (dark) {
    return (
      <span className="font-mono text-[11px] text-white/30 border border-white/[0.09] px-2.5 py-1 rounded">
        {children}
      </span>
    )
  }

  return (
    <span className="font-mono text-[11px] text-[#555] bg-[#f4f4f4] border border-[#e8e8e8] px-2.5 py-1 rounded">
      {children}
    </span>
  )
}

export function Logo({ variant = "light" }: { variant?: "light" | "dark" }) {
  return (
    <div className="flex items-center gap-3">
      <img
        src="/bulb.svg"
        alt="Luumen Logo"
        width={24}
        height={24}
        style={{ filter: variant === "light" ? "invert(1)" : "none" }}
      />
      <span className={`font-bold text-[19px] tracking-tight ${variant === "light" ? "text-white" : "text-[#0a0a0a]"}`}>
        Luumen
      </span>
      <span
        className={`font-mono text-[12px] ${
          variant === "light"
            ? "text-white/60 bg-white/10 border-white/20"
            : "text-[#999] bg-[#f4f4f4] border-[#e5e5e5]"
        } px-2 py-0.5 rounded-md border`}
      >
        luu
      </span>
    </div>
  )
}
