export function TrustedStack() {
  const tools = [
    { name: "Rojo", desc: "Filesystem sync to Studio" },
    { name: "Wally", desc: "Package manager for Roblox" },
    { name: "Rokit", desc: "Toolchain version manager" },
    { name: "StyLua", desc: "Lua code formatter" },
    { name: "Selene", desc: "Lua static analyzer" },
  ]

  return (
    <section className="border-b border-[#ebebeb]">
      <div className="grid grid-cols-1 md:grid-cols-2">
        <div className="px-16 py-36 md:border-r border-b md:border-b-0 border-[#ebebeb]">
          <p className="text-[10px] font-mono text-[#bbb] uppercase tracking-[0.15em] mb-6">ECOSYSTEM</p>
          <h2 className="font-display text-[44px] font-bold text-[#0a0a0a] leading-tight mb-7">A trusted stack to standardize on</h2>
          <p className="text-[#666] text-[18px] leading-relaxed">
            Luumen is built on the established open source tools the Roblox community already uses. It composes them - it does not replace them.
          </p>
        </div>
        <div className="px-16 py-36">
          <p className="text-[10px] font-mono text-[#bbb] uppercase tracking-[0.15em] mb-6">TOOLS LUUMEN WRAPS</p>
          <div className="border border-[#ebebeb] rounded-xl overflow-hidden divide-y divide-[#ebebeb]">
            {tools.map((tool) => (
              <div key={tool.name} className="flex items-center justify-between px-6 py-5">
                <span className="font-mono text-[16px] text-[#0a0a0a] font-medium">{tool.name}</span>
                <span className="text-[#999] text-[15px]">{tool.desc}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
