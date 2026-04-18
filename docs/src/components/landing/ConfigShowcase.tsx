import { CONFIG_SOURCE } from "./data"
import { TerminalWindow } from "./shared"

function ConfigLine({ line }: { line: string }) {
  if (line === "") return <div className="leading-[1.75]">&nbsp;</div>

  const trimmed = line.trimStart()
  const indent = line.slice(0, line.length - trimmed.length)

  // Closing braces: `}`, `},`
  if (trimmed === "}" || trimmed === "},") {
    return (
      <div className="leading-[1.75] whitespace-pre">
        {indent}<span className="text-white/50">{trimmed}</span>
      </div>
    )
  }

  // `return {`
  if (trimmed === "return {") {
    return (
      <div className="leading-[1.75] whitespace-pre">
        {indent}<span className="text-rose-400">return</span><span className="text-white/50"> {"{"}</span>
      </div>
    )
  }

  // Array string items: `"...",`
  if (trimmed.startsWith('"')) {
    return (
      <div className="leading-[1.75] whitespace-pre">
        {indent}<span className="text-emerald-400">{trimmed}</span>
      </div>
    )
  }

  // `key = value` lines
  const eq = trimmed.indexOf(" = ")
  if (eq !== -1) {
    const key = trimmed.slice(0, eq)
    const val = trimmed.slice(eq + 3)

    const valNode =
      val === "{" ? (
        <span className="text-white/50">{"{"}</span>
      ) : (
        <span className="text-emerald-400">{val}</span>
      )

    return (
      <div className="leading-[1.75] whitespace-pre">
        {indent}<span className="text-sky-300">{key}</span><span className="text-white/25"> = </span>{valNode}
      </div>
    )
  }

  return <div className="leading-[1.75] whitespace-pre text-white/50">{line}</div>
}

export function ConfigShowcase() {
  return (
    <section className="border-t border-b border-[#ebebeb]">
      <div className="grid grid-cols-1 md:grid-cols-2">
        <div className="px-16 py-36 md:border-r border-b md:border-b-0 border-[#ebebeb]">
          <p className="text-[10px] font-mono text-[#bbb] uppercase tracking-[0.15em] mb-6">PROJECT.CONFIG.LUAU</p>
          <h2 className="font-display text-[44px] font-bold text-[#0a0a0a] leading-tight mb-7">A new standard for configuring Luau projects</h2>
          <p className="text-[#666] text-[18px] leading-relaxed mb-4">
            <code className="font-mono text-[13px] bg-[#f4f4f4] border border-[#e5e5e5] text-[#333] px-1.5 py-0.5 rounded">project.config.luau</code>{" "}
            is a shared Luau-native project config designed for tools across the ecosystem.
          </p>
          <p className="text-[#666] text-[17px] leading-relaxed mb-10">
            Instead of spreading project setup across multiple config files and formats, tools can read from one file written in Luau tables — making configuration easier to understand, extend, and keep in one place.
          </p>
          <ul className="space-y-5">
            {[
              "One file for commands, tasks, and project-level tooling",
              "Written in Luau tables — native to the ecosystem",
              "Extensible by multiple tools, not just Luumen",
              "Designed to keep project configuration in one place",
            ].map((item) => (
              <li key={item} className="flex items-start gap-3 text-[17px] text-[#555] leading-relaxed">
                <span className="text-rose-500 mt-0.5 shrink-0">✓</span>
                {item}
              </li>
            ))}
          </ul>
        </div>

        <div className="px-16 py-36 flex items-center">
          <TerminalWindow title="project.config.luau" className="w-full">
            <div className="px-6 py-5 font-mono text-[15px]">
              {CONFIG_SOURCE.split("\n").map((line, i) => (
                <ConfigLine key={i} line={line} />
              ))}
            </div>
          </TerminalWindow>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 border-t border-[#ebebeb]">
        <div className="px-16 py-16 md:border-r border-b md:border-b-0 border-[#ebebeb]">
          <h3 className="font-display text-[22px] font-bold text-[#0a0a0a] mb-4">Why this matters</h3>
          <p className="text-[#666] text-[17px] leading-relaxed mb-4">
            A single project config makes the Luau toolchain easier to work with.
          </p>
          <p className="text-[#666] text-[17px] leading-relaxed">
            Instead of each tool introducing its own format, tools can share one project definition and extend it with their own sections when needed. That means less fragmentation, less duplicated config, and a cleaner developer experience.
          </p>
        </div>

        <div className="px-16 py-16">
          <h3 className="font-display text-[22px] font-bold text-[#0a0a0a] mb-4">Built for the ecosystem</h3>
          <p className="text-[#666] text-[17px] leading-relaxed mb-4">
            Luumen reads <code className="font-mono text-[13px] bg-[#f4f4f4] border border-[#e5e5e5] text-[#333] px-1.5 py-0.5 rounded">project.config.luau</code>, but the format is not meant to be Luumen-only.
          </p>
          <p className="text-[#666] text-[17px] leading-relaxed">
            Other tools — for example Selene or future Luau tooling — can adopt the same file and keep project configuration in one place instead of scattering it across separate configs.
          </p>
        </div>
      </div>
    </section>
  )
}
