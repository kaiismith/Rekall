import { createTheme } from '@mui/material/styles'

/**
 * Rekall dark theme — mirrors the tailwind.config.ts color palette
 * so MUI components and Tailwind utilities stay visually consistent.
 */
const theme = createTheme({
  palette: {
    mode: 'dark',
    background: {
      default: '#0d0f14',
      paper: '#161922',
    },
    primary: {
      main: '#3b82f6',
      light: '#60a5fa',
      dark: '#2563eb',
      contrastText: '#ffffff',
    },
    secondary: {
      main: '#8b5cf6',
      light: '#a78bfa',
      dark: '#7c3aed',
      contrastText: '#ffffff',
    },
    error: {
      main: '#ef4444',
    },
    warning: {
      main: '#f59e0b',
    },
    info: {
      main: '#3b82f6',
    },
    success: {
      main: '#22c55e',
    },
    text: {
      primary: '#e2e8f0',
      secondary: '#94a3b8',
      disabled: '#475569',
    },
    divider: '#1e2330',
  },

  typography: {
    fontFamily:
      '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, system-ui, sans-serif',
    // Inter features: ss01 (single-story a), cv11 (single-story g),
    // tnum (tabular numerals), calt (contextual alternates).
    // These are applied globally via MuiCssBaseline below so EVERY surface
    // — including inputs, tables, chips — gets the enterprise typographic
    // refinements, not just headings.
    h1: { fontWeight: 800, letterSpacing: '-0.025em', lineHeight: 1.1 },
    h2: { fontWeight: 800, letterSpacing: '-0.02em', lineHeight: 1.15 },
    h3: { fontWeight: 700, letterSpacing: '-0.02em', lineHeight: 1.2 },
    h4: { fontWeight: 700, letterSpacing: '-0.015em', lineHeight: 1.25 },
    h5: { fontWeight: 600, letterSpacing: '-0.01em', lineHeight: 1.3 },
    h6: { fontWeight: 600, letterSpacing: '-0.005em', lineHeight: 1.4 },
    subtitle1: { fontWeight: 600, letterSpacing: '-0.005em' },
    subtitle2: { fontWeight: 600, letterSpacing: '0' },
    body1: { fontSize: '0.9375rem', lineHeight: 1.6, letterSpacing: '-0.003em' },
    body2: { fontSize: '0.875rem', lineHeight: 1.55, letterSpacing: '-0.002em' },
    button: { fontWeight: 600, letterSpacing: '-0.005em' },
    caption: { fontSize: '0.75rem', color: '#94a3b8', letterSpacing: '0' },
    overline: {
      letterSpacing: '0.12em',
      fontWeight: 700,
      fontSize: '0.6875rem',
      lineHeight: 1.6,
    },
  },

  shape: {
    borderRadius: 8,
  },

  components: {
    MuiCssBaseline: {
      styleOverrides: {
        html: {
          scrollbarColor: '#1e2330 #0d0f14',
          WebkitFontSmoothing: 'antialiased',
          MozOsxFontSmoothing: 'grayscale',
          textRendering: 'optimizeLegibility',
          // Inter OpenType features — applied globally so every surface
          // picks them up (inputs, tables, chips, dialogs, etc.).
          fontFeatureSettings:
            '"ss01" on, "cv11" on, "calt" on, "case" on, "kern" on',
        },
        '&::-webkit-scrollbar': { width: '6px', height: '6px' },
        '&::-webkit-scrollbar-track': { background: '#0d0f14' },
        '&::-webkit-scrollbar-thumb': {
          background: '#1e2330',
          borderRadius: '3px',
          '&:hover': { background: '#2d3447' },
        },
        body: {
          backgroundColor: '#0d0f14',
          color: '#e2e8f0',
          fontFeatureSettings:
            '"ss01" on, "cv11" on, "calt" on, "case" on, "kern" on',
        },
        // Numerals used in tables and dashboards should align vertically.
        'table, .tabular-nums': {
          fontVariantNumeric: 'tabular-nums',
        },
        // Enterprise monospace stack for code/kbd/samp/pre.
        'code, kbd, samp, pre': {
          fontFamily:
            '"JetBrains Mono", "SF Mono", "Consolas", "Liberation Mono", Menlo, monospace',
          fontFeatureSettings: '"calt" on, "liga" on, "zero" on',
        },
        // Better selection colour to match the accent language.
        '::selection': {
          backgroundColor: 'rgba(129,140,248,0.35)',
          color: '#ffffff',
        },
      },
    },

    MuiCard: {
      styleOverrides: {
        root: {
          backgroundImage:
            'linear-gradient(145deg, rgba(255,255,255,0.025) 0%, rgba(255,255,255,0.01) 100%)',
          backgroundColor: '#161922',
          border: '1px solid rgba(255,255,255,0.06)',
          borderRadius: '12px',
          boxShadow:
            '0 1px 3px rgba(0,0,0,0.5), 0 0 0 1px rgba(255,255,255,0.02) inset',
          transition: 'border-color 160ms ease, box-shadow 160ms ease',
        },
      },
    },

    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
          border: '1px solid #1e2330',
        },
      },
    },

    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          fontWeight: 500,
          borderRadius: '6px',
        },
        contained: {
          boxShadow: 'none',
          '&:hover': { boxShadow: '0 2px 8px rgba(59,130,246,0.3)' },
        },
      },
    },

    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: '6px',
          fontWeight: 500,
          fontSize: '0.75rem',
        },
      },
    },

    MuiTableCell: {
      styleOverrides: {
        root: {
          borderBottom: '1px solid #1e2330',
        },
        head: {
          fontWeight: 600,
          color: '#94a3b8',
          fontSize: '0.75rem',
          letterSpacing: '0.05em',
          textTransform: 'uppercase',
        },
      },
    },

    MuiTextField: {
      defaultProps: {
        variant: 'outlined',
        size: 'medium',
      },
    },

    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          borderRadius: '10px',
          backgroundColor: 'rgba(255,255,255,0.02)',
          transition: 'border-color 120ms ease, background-color 120ms ease, box-shadow 120ms ease',
          '& .MuiOutlinedInput-notchedOutline': {
            borderColor: 'rgba(255,255,255,0.08)',
            transition: 'border-color 120ms ease',
          },
          '&:hover': {
            backgroundColor: 'rgba(255,255,255,0.035)',
            '& .MuiOutlinedInput-notchedOutline': {
              borderColor: 'rgba(255,255,255,0.16)',
            },
          },
          '&.Mui-focused': {
            backgroundColor: 'rgba(255,255,255,0.04)',
            boxShadow: '0 0 0 3px rgba(129,140,248,0.18)',
            '& .MuiOutlinedInput-notchedOutline': {
              borderColor: '#818cf8',
              borderWidth: '1px',
            },
          },
          '&.Mui-error': {
            '& .MuiOutlinedInput-notchedOutline': { borderColor: '#ef4444' },
            '&.Mui-focused': {
              boxShadow: '0 0 0 3px rgba(239,68,68,0.18)',
            },
          },
          '&.Mui-disabled': {
            backgroundColor: 'rgba(255,255,255,0.015)',
            '& .MuiOutlinedInput-notchedOutline': {
              borderColor: 'rgba(255,255,255,0.04)',
            },
          },
        },
        input: {
          padding: '12px 14px',
          fontSize: '0.9375rem',
          color: '#e2e8f0',
          caretColor: '#a78bfa',
          // Hide Edge/IE's native password-reveal button and Chrome's built-in
          // "view password" control so only our PasswordField's eye icon shows.
          '&::-ms-reveal, &::-ms-clear': { display: 'none' },
          '&::-webkit-credentials-auto-fill-button, &::-webkit-caps-lock-indicator, &::-webkit-password-toggle-button':
            { display: 'none !important', visibility: 'hidden !important' },
          // Kill Chrome/Edge autofill's yellow/blue flash and force the dark fill.
          '&:-webkit-autofill, &:-webkit-autofill:hover, &:-webkit-autofill:focus, &:-webkit-autofill:active': {
            WebkitTextFillColor: '#e2e8f0',
            WebkitBoxShadow: '0 0 0 1000px #151823 inset',
            caretColor: '#a78bfa',
            borderRadius: 'inherit',
            transition: 'background-color 5000s ease-in-out 0s',
          },
          '&::placeholder': {
            color: '#64748b',
            opacity: 1,
          },
        },
      },
    },

    MuiInputLabel: {
      styleOverrides: {
        root: {
          color: '#94a3b8',
          fontSize: '0.875rem',
          '&.Mui-focused': { color: '#a78bfa' },
          '&.Mui-error': { color: '#ef4444' },
        },
        asterisk: {
          color: '#64748b',
        },
      },
    },

    MuiFormHelperText: {
      styleOverrides: {
        root: {
          marginLeft: 2,
          marginTop: 6,
          fontSize: '0.75rem',
          color: '#64748b',
          '&.Mui-error': { color: '#ef4444' },
        },
      },
    },

    MuiInputAdornment: {
      styleOverrides: {
        root: {
          color: '#64748b',
        },
      },
    },

    MuiSelect: {
      styleOverrides: {
        icon: { color: '#64748b' },
      },
    },

    MuiMenu: {
      styleOverrides: {
        paper: {
          backgroundColor: '#151823',
          border: '1px solid rgba(255,255,255,0.06)',
          borderRadius: '10px',
          boxShadow: '0 10px 40px rgba(0,0,0,0.45)',
          marginTop: 4,
        },
      },
    },

    MuiMenuItem: {
      styleOverrides: {
        root: {
          fontSize: '0.9375rem',
          padding: '10px 14px',
          '&.Mui-selected': {
            backgroundColor: 'rgba(129,140,248,0.14)',
            '&:hover': { backgroundColor: 'rgba(129,140,248,0.2)' },
          },
        },
      },
    },

    MuiAlert: {
      styleOverrides: {
        root: {
          borderRadius: '10px',
          border: '1px solid transparent',
        },
        standardError: {
          backgroundColor: 'rgba(239,68,68,0.08)',
          borderColor: 'rgba(239,68,68,0.25)',
          color: '#fca5a5',
        },
        standardSuccess: {
          backgroundColor: 'rgba(34,197,94,0.08)',
          borderColor: 'rgba(34,197,94,0.25)',
          color: '#86efac',
        },
        standardInfo: {
          backgroundColor: 'rgba(59,130,246,0.08)',
          borderColor: 'rgba(59,130,246,0.25)',
          color: '#93c5fd',
        },
        standardWarning: {
          backgroundColor: 'rgba(245,158,11,0.08)',
          borderColor: 'rgba(245,158,11,0.25)',
          color: '#fcd34d',
        },
      },
    },

    MuiLink: {
      styleOverrides: {
        root: {
          color: '#a78bfa',
          textDecorationColor: 'rgba(167,139,250,0.35)',
          transition: 'color 120ms ease, text-decoration-color 120ms ease',
          '&:hover': {
            color: '#c4b5fd',
            textDecorationColor: '#c4b5fd',
          },
        },
      },
    },

    MuiTooltip: {
      styleOverrides: {
        tooltip: {
          backgroundColor: '#1e2330',
          border: '1px solid #2d3447',
          fontSize: '0.75rem',
        },
      },
    },

    MuiDrawer: {
      styleOverrides: {
        paper: {
          backgroundColor: '#0d0f14',
          borderRight: '1px solid #1e2330',
        },
      },
    },

    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundColor: '#0d0f14',
          borderBottom: '1px solid #1e2330',
          boxShadow: 'none',
        },
      },
    },

    MuiListItemButton: {
      styleOverrides: {
        root: {
          borderRadius: '6px',
          '&.Mui-selected': {
            backgroundColor: 'rgba(59,130,246,0.15)',
            '&:hover': { backgroundColor: 'rgba(59,130,246,0.2)' },
          },
        },
      },
    },
  },
})

export default theme

// ─── Design tokens ────────────────────────────────────────────────────────────
//
// Named export consumed by the `common/ui` component library. Gradients and
// glow shadows live here so the new visual language has a single source of
// truth. Do not inline these literals elsewhere.

export interface Tokens {
  gradients: {
    primary: string
    primaryHover: string
    heroBackground: string
    cardSurface: string
  }
  shadows: {
    glowPrimary: string
    glowPrimaryHover: string
    elevatedCard: string
  }
  radii: {
    card: number
    button: number
    keyChip: number
  }
  fonts: {
    sans: string
    mono: string
  }
}

export const tokens: Tokens = {
  gradients: {
    primary: 'linear-gradient(135deg, #a78bfa 0%, #818cf8 50%, #60a5fa 100%)',
    primaryHover: 'linear-gradient(135deg, #b9a2fc 0%, #93a0fb 50%, #7cb5fc 100%)',
    heroBackground:
      'radial-gradient(ellipse 80% 60% at 50% 0%, rgba(139,92,246,0.12) 0%, transparent 60%)',
    cardSurface: 'linear-gradient(145deg, #1a1e2b 0%, #13161f 100%)',
  },
  shadows: {
    glowPrimary: '0 0 32px rgba(129,140,248,0.35), 0 4px 20px rgba(0,0,0,0.4)',
    glowPrimaryHover: '0 0 40px rgba(129,140,248,0.5), 0 4px 24px rgba(0,0,0,0.45)',
    elevatedCard: '0 10px 40px rgba(0,0,0,0.45), 0 0 0 1px rgba(255,255,255,0.04)',
  },
  radii: { card: 12, button: 10, keyChip: 4 },
  fonts: {
    sans: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, system-ui, sans-serif',
    mono: '"JetBrains Mono", "SF Mono", "Consolas", "Liberation Mono", Menlo, monospace',
  },
}
