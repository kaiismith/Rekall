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
    fontFamily: '"Inter", system-ui, sans-serif',
    h1: { fontWeight: 700 },
    h2: { fontWeight: 700 },
    h3: { fontWeight: 600 },
    h4: { fontWeight: 600 },
    h5: { fontWeight: 600 },
    h6: { fontWeight: 600 },
    body1: { fontSize: '0.9375rem', lineHeight: 1.6 },
    body2: { fontSize: '0.875rem', lineHeight: 1.5 },
    caption: { fontSize: '0.75rem', color: '#94a3b8' },
    overline: { letterSpacing: '0.1em', fontWeight: 600 },
  },

  shape: {
    borderRadius: 8,
  },

  components: {
    MuiCssBaseline: {
      styleOverrides: {
        html: { scrollbarColor: '#1e2330 #0d0f14' },
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
        },
      },
    },

    MuiCard: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
          backgroundColor: '#161922',
          border: '1px solid #1e2330',
          boxShadow: '0 1px 3px rgba(0,0,0,0.4)',
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
        size: 'small',
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
