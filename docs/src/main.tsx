import { StrictMode } from "react"
import { createRoot } from "react-dom/client"

import "./index.css"
import App from "./App.tsx"

// Force light mode variables — clear any stored theme preference
localStorage.removeItem("theme")
document.documentElement.classList.remove("dark")
document.documentElement.classList.add("light")

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
)
