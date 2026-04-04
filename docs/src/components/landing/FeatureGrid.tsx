import { FEATURES } from "./data"
import { Pill } from "./shared"

export function FeatureGrid() {
  return (
    <section id="features" className="border-b border-[#ebebeb]">
      <div className="grid grid-cols-1 md:grid-cols-3 divide-y md:divide-y-0 md:divide-x divide-[#ebebeb]">
        {FEATURES.map((feature) => (
          <div key={feature.title} className="px-16 py-32">
            <h3 className="font-display text-[26px] font-bold text-[#0a0a0a] mb-5 leading-snug">{feature.title}</h3>
            <p className="text-[#666] text-[17px] leading-relaxed mb-10">{feature.desc}</p>
            <div className="flex flex-wrap gap-2.5">
              {feature.pills.map((pill) => (
                <Pill key={pill}>{pill}</Pill>
              ))}
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}
