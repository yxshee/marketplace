import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        canvas: "var(--canvas)",
        surface: "var(--surface)",
        "surface-soft": "var(--surface-soft)",
        border: "var(--border)",
        "border-strong": "var(--border-strong)",
        line: "var(--border)",
        ink: "var(--ink)",
        muted: "var(--muted)",
        accent: "var(--accent)",
        success: "var(--success-500)",
        warning: "var(--warning-500)",
        danger: "var(--danger-500)",
        brand: {
          50: "var(--brand-50)",
          100: "var(--brand-100)",
          200: "var(--brand-200)",
          300: "var(--brand-300)",
          400: "var(--brand-400)",
          500: "var(--brand-500)",
          600: "var(--brand-600)",
          700: "var(--brand-700)",
        },
        indigo: {
          100: "var(--accent-100)",
          300: "var(--accent-300)",
          500: "var(--accent-500)",
          600: "var(--accent-600)",
        },
      },
      borderRadius: {
        xs: "var(--radius-xs)",
        sm: "var(--radius-sm)",
        md: "var(--radius-md)",
        lg: "var(--radius-lg)",
      },
      boxShadow: {
        soft: "var(--shadow-soft)",
        crisp: "var(--shadow-card)",
        float: "var(--shadow-float)",
        glow: "var(--shadow-glow)",
      },
      fontFamily: {
        display: ["\"Sora\"", "ui-sans-serif", "system-ui", "sans-serif"],
        body: ["\"Manrope\"", "ui-sans-serif", "system-ui", "sans-serif"],
      },
      fontSize: {
        "display-1": ["clamp(2.2rem, 1.8rem + 1.3vw, 3.4rem)", { lineHeight: "1.06", fontWeight: "700" }],
        "display-2": ["clamp(1.85rem, 1.55rem + 0.85vw, 2.6rem)", { lineHeight: "1.1", fontWeight: "650" }],
        "display-3": ["clamp(1.45rem, 1.28rem + 0.45vw, 1.95rem)", { lineHeight: "1.15", fontWeight: "650" }],
      },
      backgroundImage: {
        "brand-gradient": "linear-gradient(135deg, var(--brand-500), var(--accent-500))",
        "hero-wash":
          "radial-gradient(circle at 18% 12%, rgba(255, 148, 206, 0.4), transparent 35%), radial-gradient(circle at 86% 8%, rgba(137, 152, 255, 0.36), transparent 28%)",
      },
      spacing: {
        18: "4.5rem",
        22: "5.5rem",
        30: "7.5rem",
      },
    },
  },
  plugins: [],
};

export default config;
