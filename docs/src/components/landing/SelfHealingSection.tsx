import { lineClass, TerminalWindow } from "./shared"
import type { OutputLine } from "./types"

const LINES: OutputLine[] = [
  { text: "  [luu] workspace: my-game", kind: "muted" },
  { text: "  [luu] task: lint", kind: "muted" },
  { text: "  [luu] running: selene src", kind: "plain" },
  { text: "", kind: "blank" },
  { text: "  [luu] Missing tool: selene", kind: "accent" },
  { text: "  [luu] Install with Rokit now? [Y/n]: y", kind: "accent" },
  { text: "", kind: "blank" },
  { text: "  [luu] running: rokit add kampfkarren/selene", kind: "plain" },
  { text: "  [ok] Tool installed", kind: "success" },
  { text: "", kind: "blank" },
  { text: "  [luu] running: selene src", kind: "plain" },
  { text: "", kind: "blank" },
  { text: "  Results:", kind: "plain" },
  { text: "  0 errors", kind: "success" },
  { text: "  0 warnings", kind: "success" },
  { text: "  0 parse errors", kind: "success" },
]

export function SelfHealingSection() {
  return (
    <section className="border-b border-[#ebebeb]">
      <div className="grid grid-cols-1 md:grid-cols-2">
        <div className="px-16 py-36 md:border-r border-b md:border-b-0 border-[#ebebeb]">
          <p className="text-[10px] font-mono text-[#bbb] uppercase tracking-[0.15em] mb-6">SELF-HEALING</p>
          <h2 className="font-display text-[44px] font-bold text-[#0a0a0a] leading-tight mb-7">Self-healing workflows</h2>
          <p className="text-[#666] text-[18px] leading-relaxed mb-10">
            Run any task and Luumen installs missing tools automatically, then continues execution.
          </p>
          <ul className="space-y-5">
            {[
              "Tasks never fail just because a tool isn't installed",
              "Prompts once, then resumes without interruption",
              "Uses Rokit to install — no manual setup required",
              "Works for any tool defined in your project",
            ].map((item) => (
              <li key={item} className="flex items-start gap-3 text-[17px] text-[#555] leading-relaxed">
                <span className="text-rose-500 mt-0.5 shrink-0">✓</span>
                {item}
              </li>
            ))}
          </ul>
        </div>

        <div className="px-16 py-36 flex items-center">
          <TerminalWindow className="w-full">
            <div className="px-7 py-6 font-mono text-[15px] leading-[1.85]">
              <div className="mb-2.5">
                <span className="text-white/25">$ </span>
                <span className="text-white font-medium">luu lint</span>
              </div>
              {LINES.map((line, i) => (
                <div key={i} className={lineClass(line.kind)}>
                  {line.kind === "blank" ? "\u00A0" : line.text}
                </div>
              ))}
            </div>
          </TerminalWindow>
        </div>
      </div>
    </section>
  )
}
