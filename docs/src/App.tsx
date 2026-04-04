import { AdoptionSection } from "./components/landing/AdoptionSection"
import { CommandsShowcase } from "./components/landing/CommandsShowcase"
import { ConfigShowcase } from "./components/landing/ConfigShowcase"
import { CTASection } from "./components/landing/CTASection"
import { FeatureGrid } from "./components/landing/FeatureGrid"
import { Footer } from "./components/landing/Footer"
import { GettingStarted } from "./components/landing/GettingStarted"
import { Hero } from "./components/landing/Hero"
import { Navbar } from "./components/landing/Navbar"
import { OpenSourceSection } from "./components/landing/OpenSourceSection"
import { TrustedStack } from "./components/landing/TrustedStack"

export default function App() {
  return (
    <div
      className="relative min-h-screen mx-auto bg-white"
      style={{
        maxWidth: 1440,
        borderLeft: "1px solid rgba(128,128,128,0.38)",
        borderRight: "1px solid rgba(128,128,128,0.38)",
      }}
    >
      <div className="pointer-events-none absolute inset-y-0 left-0 z-[60] w-px bg-[rgba(128,128,128,0.42)]" />
      <div className="pointer-events-none absolute inset-y-0 right-0 z-[60] w-px bg-[rgba(128,128,128,0.42)]" />
      <Navbar />
      <main>
        <Hero />
        <GettingStarted />
        <FeatureGrid />
        <CommandsShowcase />
        <ConfigShowcase />
        <AdoptionSection />
        <TrustedStack />
        <OpenSourceSection />
        <CTASection />
      </main>
      <Footer />
    </div>
  )
}
