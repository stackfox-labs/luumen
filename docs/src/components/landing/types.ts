export type TabId = "create" | "install" | "add" | "dev" | "lint" | "doctor"

export interface OutputLine {
  text: string
  kind: "success" | "accent" | "muted" | "plain" | "blank"
}

export interface CommandSection {
  id: TabId
  label: string
  command: string
  heading: string
  sub: string
  points: string[]
  lines: OutputLine[]
}
