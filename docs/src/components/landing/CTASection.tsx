export function CTASection() {
  return (
    <section className="bg-[#080808] shell-bleed-dark">
      <div
        className="relative overflow-hidden"
        style={{
          background:
            "radial-gradient(ellipse 70% 85% at 50% 0%, rgba(244,63,94,0.25) 0%, rgba(190,18,60,0.08) 50%, transparent 78%), #080808",
        }}
      >
        <div
          className="absolute inset-0 pointer-events-none"
          style={{
            backgroundImage: "linear-gradient(rgba(0,0,0,0.5), rgba(0,0,0,0.5)), url('/Banner2.png')",
            backgroundSize: "cover",
            backgroundPosition: "center",
            backgroundRepeat: "no-repeat",
          }}
        />
        <div className="relative z-10 px-8 py-32 md:py-36 text-center max-w-3xl mx-auto">
          <h2 className="font-display text-[52px] md:text-[62px] font-bold text-white leading-[1.06] mb-6 tracking-[-0.01em]">
            Start building better
            <br />
            Roblox repos today
          </h2>
          <a
            href="#start"
            className="inline-flex items-center gap-2 bg-white text-[#0a0a0a] text-sm font-semibold px-7 py-3.5 rounded-xl hover:bg-white/90 transition-colors"
          >
            Get started →
          </a>
        </div>
      </div>
    </section>
  )
}
