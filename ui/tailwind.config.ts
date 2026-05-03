import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        shell: "#0b0e11",
        panel: "#1e2329",
        elevated: "#2b3139",
        accent: "#fcd535",
        accentActive: "#f0b90b",
        body: "#eaecef",
        muted: "#707a8a",
        ink: "#181a20",
        line: "#2b3139",
        up: "#0ecb81",
        down: "#f6465d",
      },
      borderRadius: {
        ui: "8px",
        panel: "12px",
      },
      boxShadow: {
        panel: "0 18px 48px rgba(0, 0, 0, 0.28)",
      },
      fontFamily: {
        sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'sans-serif'],
        numeric: ['"IBM Plex Sans"', '"JetBrains Mono"', 'monospace'],
      },
      letterSpacing: {
        tightest: "-0.06em",
      },
    },
  },
  plugins: [],
} satisfies Config;
