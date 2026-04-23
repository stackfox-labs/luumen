import { TerminalWindow } from "./shared"

export function AdoptionSection() {
  return (
    <section className="border-b border-[#ebebeb]">
      <div className="grid grid-cols-1 md:grid-cols-2">
        <div className="px-16 py-36 md:border-r border-b md:border-b-0 border-[#ebebeb]">
          <h3 className="font-display text-[32px] font-bold text-[#0a0a0a] mb-5">Starting a new project?</h3>
          <p className="text-[#666] text-[17px] leading-relaxed mb-10">
            One command scaffolds a complete repo with all config files - ready to develop immediately.
          </p>
          <TerminalWindow>
            <div className="px-6 py-5 font-mono text-[15px] leading-[1.9]">
              <div>
                <span className="text-white/25">$ </span>
                <span className="text-white">luu create my-game</span>
              </div>
              <div>
                <span className="text-white/25">$ </span>
                <span className="text-white">cd my-game</span>
              </div>
              <div>
                <span className="text-white/25">$ </span>
                <span className="text-white">luu dev</span>
              </div>
            </div>
          </TerminalWindow>
        </div>

        <div className="px-16 py-36">
          <h3 className="font-display text-[32px] font-bold text-[#0a0a0a] mb-5">Adopting an existing repo?</h3>
          <p className="text-[#666] text-[17px] leading-relaxed mb-10">
            <code className="font-mono text-[13px] text-[#555]">luu init</code> detects your existing Rojo, Wally, and Rokit config and generates a{" "}
            <code className="font-mono text-[13px] text-[#555]">.config.luau</code> to match.
          </p>
          <TerminalWindow>
            <div className="px-6 py-5 font-mono text-[15px] leading-[1.9]">
              <div>
                <span className="text-white/25">$ </span>
                <span className="text-white">luu init</span>
              </div>
              <div>
                <span className="text-white/25">$ </span>
                <span className="text-white">luu install</span>
              </div>
              <div>
                <span className="text-white/25">$ </span>
                <span className="text-white">luu dev</span>
              </div>
            </div>
          </TerminalWindow>
        </div>
      </div>
    </section>
  )
}
