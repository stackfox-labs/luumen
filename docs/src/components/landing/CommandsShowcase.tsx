import { useEffect, useRef, useState } from "react"

import { SECTIONS } from "./data"
import { TAB_ICONS } from "./icons"
import { lineClass } from "./shared"
import type { TabId } from "./types"

function poweredByForSection(id: TabId): string[] {
  switch (id) {
    case "create":
    case "install":
      return ["Rokit", "Wally"]
    case "add":
      return ["Rokit", "Wally"]
    case "dev":
      return ["Rojo"]
    case "lint":
      return ["Selene", "StyLua"]
    case "doctor":
      return ["Luumen", "Rokit", "Rojo"]
    default:
      return ["your shell"]
  }
}

export function CommandsShowcase() {
  const [activeId, setActiveId] = useState<TabId>("create")
  const sectionRefs = useRef<Partial<Record<TabId, HTMLDivElement>>>({})
  const stickyRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleScroll = () => {
      const threshold = window.innerHeight * 0.45
      let best: TabId = SECTIONS[0].id

      for (const section of SECTIONS) {
        const el = sectionRefs.current[section.id]
        if (!el) continue
        const top = el.getBoundingClientRect().top
        if (top <= threshold) best = section.id
      }

      setActiveId(best)
    }

    window.addEventListener("scroll", handleScroll, { passive: true })
    return () => window.removeEventListener("scroll", handleScroll)
  }, [])

  const scrollTo = (id: TabId) => {
    const el = sectionRefs.current[id]
    if (!el) return

    const y = el.getBoundingClientRect().top + window.scrollY - 84
    window.scrollTo({ top: y, behavior: "smooth" })
  }

  return (
    <section id="commands" className="bg-[#080808] shell-bleed-dark">
      <div className="pt-40 pb-20 text-center px-16">
        <h2 className="font-display text-[54px] font-bold text-white leading-tight mb-6">Everything you need in one tool</h2>
        <p className="text-white/35 text-[20px] max-w-[520px] mx-auto leading-relaxed">
          Luumen unifies your entire Roblox workflow into a single, consistent command surface.
        </p>
      </div>

      <div
        ref={stickyRef}
        className="sticky z-30 bg-[#080808]/95 border-b border-white/[0.08]"
        style={{ top: 0, backdropFilter: "blur(12px)" }}
      >
        <div className="flex">
          {SECTIONS.map((section) => (
            <button
              key={section.id}
              onClick={() => scrollTo(section.id)}
              className={`relative flex-1 flex flex-col items-center gap-2 py-5 cursor-pointer transition-all ${
                activeId === section.id ? "text-white" : "text-white/28 hover:text-white/55"
              }`}
            >
              {(() => {
                const Icon = TAB_ICONS[section.id]
                return <Icon active={activeId === section.id} />
              })()}
              <span className={`font-mono text-[12px] tracking-wide uppercase transition-colors ${activeId === section.id ? "text-white" : "text-white/28"}`}>
                {section.label}
              </span>
              {activeId === section.id && <span className="absolute bottom-0 left-0 right-0 h-[2px] bg-rose-500" />}
            </button>
          ))}
        </div>
      </div>

      {SECTIONS.map((section, idx) => (
        <div
          key={section.id}
          ref={(el) => {
            if (el) sectionRefs.current[section.id] = el
          }}
          className="border-b border-white/[0.06] last:border-b-0"
        >
          <div className="grid grid-cols-1 md:grid-cols-2">
            <div className="px-16 py-36 md:pr-16 md:border-r border-white/[0.06]">
              <p className="font-mono text-[10px] text-rose-500/55 tracking-[0.15em] uppercase mb-5">{section.sub}</p>
              <h3 className="font-display text-[38px] font-bold text-white mb-8 leading-tight">{section.heading}</h3>
              <ul className="space-y-4 mb-10">
                {section.points.map((point) => (
                  <li key={point} className="flex items-start gap-3 text-[17px] text-white/45 leading-relaxed">
                    <span className="text-rose-500 shrink-0 mt-0.5">✓</span>
                    {point}
                  </li>
                ))}
              </ul>
              <p className="text-white/20 text-xs font-mono">
                Powered by{" "}
                {poweredByForSection(section.id).map((tool, toolIndex) => (
                  <span key={tool}>
                    {toolIndex > 0 ? " & " : ""}
                    <span className="text-white/35">{tool}</span>
                  </span>
                ))}
              </p>
            </div>

            <div className="px-16 pt-24 pb-0 md:pl-16 flex flex-col">
              <div
                className="flex-1 flex flex-col overflow-hidden"
                style={{
                  borderRadius: "16px 16px 0 0",
                  padding: "1px 1px 0 1px",
                  background: [
                    "linear-gradient(135deg, rgba(244,63,94,0.35) 0%, rgba(255,255,255,0.06) 60%)",
                    "linear-gradient(135deg, rgba(244,63,94,0.28) 0%, rgba(255,255,255,0.05) 60%)",
                    "linear-gradient(135deg, rgba(16,185,129,0.28) 0%, rgba(255,255,255,0.05) 60%)",
                    "linear-gradient(135deg, rgba(251,146,60,0.30) 0%, rgba(255,255,255,0.05) 60%)",
                    "linear-gradient(135deg, rgba(244,63,94,0.32) 0%, rgba(255,255,255,0.06) 60%)",
                    "linear-gradient(135deg, rgba(14,165,233,0.30) 0%, rgba(255,255,255,0.05) 60%)",
                    "linear-gradient(135deg, rgba(245,158,11,0.30) 0%, rgba(255,255,255,0.05) 60%)",
                  ][idx],
                  boxShadow: "0 -24px 80px rgba(0,0,0,0.5), 0 -4px 24px rgba(244,63,94,0.08)",
                }}
              >
                <div className="flex-1 flex flex-col bg-[#0d0d0d] overflow-hidden" style={{ borderRadius: "15px 15px 0 0" }}>
                  <div className="px-7 pt-7 pb-6 font-mono text-[15px] leading-[1.85] overflow-hidden">
                    <div className="mb-2.5">
                      <span className="text-white/25">$ </span>
                      <span className="text-white font-medium">{section.command}</span>
                    </div>
                    {section.lines.map((line, i) => (
                      <div key={i} className={lineClass(line.kind)}>
                        {line.kind === "blank" ? "\u00A0" : line.text}
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      ))}
    </section>
  )
}
