import { useState } from "react"

import { SECTIONS } from "./data"
import { TAB_ICONS } from "./icons"
import { lineClass } from "./shared"
import type { TabId } from "./types"

function HeroTerminal() {
  const [activeTab, setActiveTab] = useState<TabId>("create")
  const [visibleTab, setVisibleTab] = useState<TabId>("create")
  const [opacity, setOpacity] = useState(1)

  const switchTab = (id: TabId) => {
    if (id === activeTab) return
    setActiveTab(id)
    setOpacity(0)
    setTimeout(() => {
      setVisibleTab(id)
      setOpacity(1)
    }, 130)
  }

  const section = SECTIONS.find((s) => s.id === visibleTab)
  if (!section) return null

  return (
    <div
      className="flex flex-col overflow-hidden flex-1"
      style={{
        borderRadius: "16px 16px 0 0",
        padding: "1px 1px 0 1px",
        background: "linear-gradient(135deg, rgba(244,63,94,0.35) 0%, rgba(255,255,255,0.07) 55%)",
        boxShadow: "0 -20px 80px rgba(0,0,0,0.55), 0 -4px 32px rgba(244,63,94,0.1)",
      }}
    >
      <div className="flex flex-col overflow-hidden flex-1 bg-[#0c0c0c]" style={{ borderRadius: "15px 15px 0 0" }}>
        <div
          className="px-10 pt-8 font-mono text-[15px] leading-[1.85] transition-opacity"
          style={{ height: 500, overflow: "hidden", opacity, transitionDuration: "130ms" }}
        >
          <div className="mb-3">
            <span className="text-white/25">$ </span>
            <span className="text-white font-medium">{section.command}</span>
          </div>
          {section.lines.map((line, i) => (
            <div key={`${visibleTab}-${i}`} className={lineClass(line.kind)}>
              {line.kind === "blank" ? "\u00A0" : line.text}
            </div>
          ))}
        </div>

        <div className="border-t border-white/[0.07] bg-[#0a0a0a] flex">
          {SECTIONS.map((s) => (
            <button
              key={s.id}
              onClick={() => switchTab(s.id)}
              className={`relative flex-1 flex flex-col items-center gap-2 py-4 cursor-pointer transition-all ${
                activeTab === s.id ? "text-white" : "text-white/28 hover:text-white/55"
              }`}
            >
              {(() => {
                const Icon = TAB_ICONS[s.id]
                return <Icon active={activeTab === s.id} />
              })()}
              <span className={`font-mono text-[11px] tracking-wide uppercase ${activeTab === s.id ? "text-white" : "text-white/28"}`}>
                {s.label}
              </span>
              {activeTab === s.id && <span className="absolute top-0 left-0 right-0 h-[2px] bg-rose-500" />}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

export function Hero() {
  return (
    <section className="bg-white border-b border-[#ebebeb] flex flex-col">
      <div className="px-16 pt-32 pb-18 flex flex-col items-center text-center">
        <h1 className="font-display text-[64px] md:text-[80px] font-bold text-[#0a0a0a] leading-[1.03] tracking-[-0.025em] max-w-4xl mb-8">
          The unified CLI for{" "}
          <span
            className="text-transparent bg-clip-text"
            style={{
              backgroundImage: "url('/Banner1.png')",
              backgroundSize: "cover",
              backgroundPosition: "top",
              backgroundRepeat: "no-repeat",
            }}
          >
            Luau development
          </span>
        </h1>

        <p className="text-[#666] text-[19px] leading-relaxed max-w-[560px] mb-12">
          Scaffold projects, manage tools, and run workflows in one CLI.
        </p>

        <div className="flex items-center gap-3">
          <a href="#start" className="bg-[#0a0a0a] text-white text-[16px] font-semibold px-8 py-3.5 rounded-lg hover:bg-[#1f1f1f] transition-colors">
            Get started
          </a>
          <a
            href="#"
            className="text-[#444] text-[16px] font-medium px-8 py-3.5 rounded-lg border border-[#ddd] hover:border-[#bbb] hover:text-[#111] transition-colors"
          >
            Read the docs
          </a>
        </div>
      </div>

      <div className="relative flex-1 flex flex-col bg-[#090909]">
        <div
          className="absolute inset-0 pointer-events-none"
          style={{
            background:
              "radial-gradient(ellipse 70% 60% at 50% 110%, rgba(244,63,94,0.26) 0%, rgba(225,29,72,0.09) 55%, transparent 78%)",
          }}
        />
        <div
          className="absolute inset-0 pointer-events-none"
          style={{
            backgroundImage: "url('/Banner1.png')",
            backgroundSize: "cover",
            backgroundPosition: "center",
            backgroundRepeat: "no-repeat",
          }}
        />
        <div className="relative z-10 px-16 pt-32 pb-0 flex flex-col flex-1 items-center">
          <div className="w-full max-w-5xl flex flex-col flex-1">
            <HeroTerminal />
          </div>
        </div>
      </div>
    </section>
  )
}
