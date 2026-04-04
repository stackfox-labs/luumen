import { Logo } from "./shared"

export function Footer() {
  return (
    <footer className="bg-[#070707] border-t border-white/[0.05] shell-bleed-footer">
      <div className="px-16 pt-16 pb-10 flex flex-col md:flex-row justify-between gap-12">
        <div className="max-w-xs">
          <Logo variant="light" />
          <p className="text-white/25 text-sm leading-relaxed">
            The unified CLI for Roblox filesystem-first development. Built for developers who build outside Studio.
          </p>
        </div>

        <div className="flex gap-16 text-[13px]">
          <div>
            <p className="text-white/20 uppercase text-[9px] tracking-[0.16em] font-mono mb-6">PROJECT</p>
            <div className="space-y-4">
              {["GitHub", "Documentation", "Releases", "Changelog"].map((link) => (
                <div key={link}>
                  <a href="#" className="text-white/35 hover:text-white/65 transition-colors">
                    {link}
                  </a>
                </div>
              ))}
            </div>
          </div>
          <div>
            <p className="text-white/20 uppercase text-[9px] tracking-[0.16em] font-mono mb-6">COMMUNITY</p>
            <div className="space-y-4">
              {["Discord", "X / Twitter", "Issues", "Discussions"].map((link) => (
                <div key={link}>
                  <a href="#" className="text-white/35 hover:text-white/65 transition-colors">
                    {link}
                  </a>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      <div className="px-16 pb-8 border-t border-white/[0.04] pt-6">
        <p className="text-white/15 text-[11px] font-mono">© 2026 Luumen - Built for Roblox developers.</p>
      </div>
    </footer>
  )
}
