import { INSTALL_LINES } from "./data"
import { CopyButton } from "./shared"

export function GettingStarted() {
  return (
    <section id="start" className="border-b border-[#ebebeb]">
      <div className="grid grid-cols-1 md:grid-cols-2">
        <div className="px-16 py-36 md:border-r border-b md:border-b-0 border-[#ebebeb]">
          <p className="text-[10px] font-mono text-[#bbb] uppercase tracking-[0.15em] mb-6">GETTING STARTED</p>
          <h2 className="font-display text-[44px] font-bold text-[#0a0a0a] leading-tight mb-6">Install luu globally</h2>
          <p className="text-[#666] text-[18px] leading-relaxed mb-4">
            Install Luumen once, open a new terminal, then run{" "}
            <code className="font-mono text-[13px] bg-[#f4f4f4] border border-[#e5e5e5] text-[#333] px-1.5 py-0.5 rounded">luu help</code>{" "}
            in any Roblox project.
          </p>
          <p className="text-[#999] text-[16px] leading-relaxed">
            Works in any repo using Rojo, Wally, or Rokit - even without a{" "}
            <code className="font-mono text-[14px] text-[#666]">project.config.luau</code>.
          </p>
        </div>

        <div className="px-16 py-36 flex flex-col gap-5 justify-center">
          {INSTALL_LINES.map(({ label, cmd }) => (
            <div key={label} className="bg-[#0d0d0d] rounded-xl border border-white/[0.06] overflow-hidden">
              <div className="flex items-center justify-between px-5 py-3 border-b border-white/[0.06]">
                <span className="font-mono text-[10px] text-white/25 tracking-[0.12em] uppercase">{label}</span>
                <CopyButton text={cmd} />
              </div>
              <div className="px-5 py-4 font-mono text-[15px] text-white/70 overflow-x-auto whitespace-nowrap">{cmd}</div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
