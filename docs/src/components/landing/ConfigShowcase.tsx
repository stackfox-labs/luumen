import { CONFIG_SOURCE } from "./data"
import { TerminalWindow } from "./shared"

function ConfigLine({ line }: { line: string }) {
  if (line === "") return <div className="leading-[1.75]">&nbsp;</div>
  if (line.startsWith("[")) return <div className="text-rose-400 leading-[1.75]">{line}</div>

  const eq = line.indexOf(" = ")
  if (eq !== -1) {
    const key = line.slice(0, eq)
    const val = line.slice(eq + 3)
    return (
      <div className="leading-[1.75]">
        <span className="text-sky-300">{key}</span>
        <span className="text-white/25"> = </span>
        <span className="text-emerald-400">{val}</span>
      </div>
    )
  }

  return <div className="text-white/50 leading-[1.75]">{line}</div>
}

export function ConfigShowcase() {
  return (
    <section className="border-t border-b border-[#ebebeb]">
      <div className="grid grid-cols-1 md:grid-cols-2">
        <div className="px-16 py-36 md:border-r border-b md:border-b-0 border-[#ebebeb]">
          <p className="text-[10px] font-mono text-[#bbb] uppercase tracking-[0.15em] mb-6">LUUMEN.TOML</p>
          <h2 className="font-display text-[44px] font-bold text-[#0a0a0a] leading-tight mb-7">One config file for your entire workflow</h2>
          <p className="text-[#666] text-[18px] leading-relaxed mb-10">
            <code className="font-mono text-[13px] bg-[#f4f4f4] border border-[#e5e5e5] text-[#333] px-1.5 py-0.5 rounded">luumen.toml</code>{" "}
            stores workflow config that does not belong in Wally, Rokit, or Rojo - tasks, command overrides, and install behavior.
          </p>
          <ul className="space-y-5">
            {[
              "Define tasks as single commands or sequential arrays",
              "Override built-in commands per repo",
              "Control install behavior with simple flags",
              "Arrays compose sequentially - no parallel complexity",
            ].map((item) => (
              <li key={item} className="flex items-start gap-3 text-[17px] text-[#555] leading-relaxed">
                <span className="text-rose-500 mt-0.5 shrink-0">✓</span>
                {item}
              </li>
            ))}
          </ul>
        </div>

        <div className="px-16 py-36 flex items-center">
          <TerminalWindow title="luumen.toml" className="w-full">
            <div className="px-6 py-5 font-mono text-[15px]">
              {CONFIG_SOURCE.split("\n").map((line, i) => (
                <ConfigLine key={i} line={line} />
              ))}
            </div>
          </TerminalWindow>
        </div>
      </div>
    </section>
  )
}
