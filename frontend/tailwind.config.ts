import type { Config } from 'tailwindcss'

const config: Config = {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  // Let MUI handle dark mode; Tailwind is used for layout/utility classes
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Mirror the MUI dark theme palette so both systems stay in sync
        background: {
          DEFAULT: '#0d0f14',
          paper: '#161922',
          elevated: '#1e2330',
        },
        primary: {
          DEFAULT: '#3b82f6',   // blue-500
          hover: '#2563eb',     // blue-600
          muted: '#1d4ed8',     // blue-700
        },
        surface: '#161922',
        border: '#1e2330',
        muted: '#94a3b8',       // slate-400
        'text-primary': '#e2e8f0',    // slate-200
        'text-secondary': '#94a3b8',  // slate-400
        success: '#22c55e',     // green-500
        warning: '#f59e0b',     // amber-500
        error: '#ef4444',       // red-500
        info: '#3b82f6',        // blue-500
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      borderRadius: {
        DEFAULT: '8px',
      },
      boxShadow: {
        card: '0 1px 3px rgba(0,0,0,0.4), 0 1px 2px rgba(0,0,0,0.5)',
        elevated: '0 4px 6px rgba(0,0,0,0.4), 0 2px 4px rgba(0,0,0,0.3)',
      },
    },
  },
  plugins: [],
}

export default config
