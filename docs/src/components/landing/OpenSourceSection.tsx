export function OpenSourceSection() {
  return (
    <section className="bg-[#080808] shell-bleed-dark border-t border-white/[0.07] border-b border-white/[0.07]">
      <div className="grid grid-cols-1 lg:grid-cols-2 items-center gap-12 lg:gap-16 px-16 py-24 lg:py-28">
        <div className="max-w-2xl">
          <h2 className="font-display text-[56px] font-bold text-white leading-[1.04] mb-7">Free &amp; open source</h2>
          <p className="text-white/62 text-[34px] md:text-[17px] leading-relaxed mb-10 max-w-xl">
            Luumen is free and open source, built in public by developers who care about better Roblox workflows.
          </p>
          <a
            href="https://github.com/stackfox-labs/luumen"
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-2 rounded-xl border border-white/18 px-7 py-3.5 text-white/92 text-[18px] md:text-[16px] font-semibold hover:bg-white/8 transition-colors"
          >
            Contribute
          </a>
        </div>

        <div className="flex w-full flex-col md:flex-row-reverse items-center justify-center lg:justify-end gap-8 lg:justify-self-stretch">
          <img
            src="/StackFoxLogo.png"
            alt="StackFox logo"
            className="w-[112px] h-[112px] object-contain"
            draggable={false}
          />

          <div className="text-center md:text-right">
            <p className="font-mono uppercase tracking-[0.14em] text-[17px] md:text-[11px] text-white/44 mb-2">Brought to you by</p>
            <p className="font-display text-[28px] md:text-[38px] text-white font-bold tracking-tight leading-none">
              StackFox &amp; Contributors
            </p>
          </div>
        </div>
      </div>
    </section>
  )
}
