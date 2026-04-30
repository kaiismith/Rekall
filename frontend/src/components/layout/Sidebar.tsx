import { useEffect } from 'react'
import Drawer from '@mui/material/Drawer'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemButton from '@mui/material/ListItemButton'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import Box from '@mui/material/Box'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import DashboardIcon from '@mui/icons-material/Dashboard'
import PhoneInTalkIcon from '@mui/icons-material/PhoneInTalk'
import VideoCallIcon from '@mui/icons-material/VideoCall'
import BusinessIcon from '@mui/icons-material/Business'
import PersonOutlineOutlinedIcon from '@mui/icons-material/PersonOutlineOutlined'
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined'
import { useNavigate, useLocation } from 'react-router-dom'
import { SIDEBAR_WIDTH, SIDEBAR_COLLAPSED_WIDTH, ROUTES } from '@/constants'
import { useUIStore } from '@/store/uiStore'
import { useUIPreferencesStore } from '@/store/uiPreferencesStore'
import { tokens } from '@/theme'

const NAV_ITEMS = [
  { label: 'Dashboard', path: ROUTES.DASHBOARD, icon: <DashboardIcon /> },
  { label: 'Meetings', path: ROUTES.MEETINGS, icon: <VideoCallIcon /> },
  { label: 'Records', path: ROUTES.RECORDS, icon: <PhoneInTalkIcon /> },
  { label: 'Organizations', path: ROUTES.ORGANIZATIONS, icon: <BusinessIcon /> },
]

const FOOTER_NAV_ITEMS = [
  { label: 'Profile', path: ROUTES.PROFILE, icon: <PersonOutlineOutlinedIcon /> },
  { label: 'Settings', path: ROUTES.SETTINGS, icon: <SettingsOutlinedIcon /> },
]

// Module-level so the preference is applied exactly once per app lifetime,
// not once per <Sidebar/> mount. Tests that reset the uiStore in beforeEach
// can reset this too by calling `resetSidebarSeed()`.
let sidebarSeeded = false
// eslint-disable-next-line react-refresh/only-export-components
export function __resetSidebarSeedForTests() {
  sidebarSeeded = false
}

export function Sidebar() {
  const { sidebarOpen, setSidebarOpen } = useUIStore()
  const sidebarDefault = useUIPreferencesStore((s) => s.sidebarDefault)
  const navigate = useNavigate()
  const { pathname } = useLocation()

  // Seed the sidebar state from the user preference ONCE per app lifetime
  // (module-level flag) so an in-session toggle is respected thereafter.
  useEffect(() => {
    if (sidebarSeeded) return
    sidebarSeeded = true
    const want = sidebarDefault === 'expanded'
    if (want !== sidebarOpen) setSidebarOpen(want)
  }, [sidebarDefault, sidebarOpen, setSidebarOpen])

  const drawerWidth = sidebarOpen ? SIDEBAR_WIDTH : SIDEBAR_COLLAPSED_WIDTH

  return (
    <Drawer
      variant="permanent"
      sx={{
        width: drawerWidth,
        flexShrink: 0,
        transition: 'width 0.2s ease',
        '& .MuiDrawer-paper': {
          width: drawerWidth,
          overflowX: 'hidden',
          transition: 'width 0.2s ease',
          boxSizing: 'border-box',
        },
      }}
    >
      {/* Brand */}
      <Box
        sx={{
          px: sidebarOpen ? 2.5 : 1,
          py: sidebarOpen ? 2.5 : 1.75,
          display: 'flex',
          flexDirection: sidebarOpen ? 'column' : 'row',
          alignItems: sidebarOpen ? 'flex-start' : 'center',
          justifyContent: sidebarOpen ? 'flex-start' : 'center',
          gap: sidebarOpen ? 0.5 : 0,
          minHeight: 80,
          borderBottom: '1px solid rgba(255,255,255,0.05)',
          // Prevent the tagline from wrapping mid-transition (the drawer width
          // animates; without nowrap "Platform" drops to a second line until
          // the animation ends).
          whiteSpace: 'nowrap',
          overflow: 'hidden',
        }}
      >
        {sidebarOpen ? (
          <>
            <Typography
              component="span"
              sx={{
                fontSize: '1.625rem',
                fontWeight: 800,
                letterSpacing: '-0.02em',
                lineHeight: 1.1,
                backgroundImage: tokens.gradients.primary,
                WebkitBackgroundClip: 'text',
                backgroundClip: 'text',
                WebkitTextFillColor: 'transparent',
                color: 'transparent',
              }}
            >
              Rekall
            </Typography>
            <Typography
              component="span"
              sx={{
                fontSize: '0.6875rem',
                fontWeight: 700,
                letterSpacing: '0.18em',
                color: 'text.secondary',
                textTransform: 'uppercase',
                lineHeight: 1,
                whiteSpace: 'nowrap',
              }}
            >
              Intelligence Platform
            </Typography>
          </>
        ) : (
          <Box
            sx={{
              width: 32,
              height: 32,
              borderRadius: '8px',
              background: tokens.gradients.primary,
              boxShadow: '0 4px 14px rgba(129,140,248,0.4)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
            }}
          >
            <Typography
              variant="caption"
              sx={{
                fontWeight: 800,
                color: '#0a0b12',
                fontSize: '0.85rem',
                letterSpacing: '-0.02em',
              }}
            >
              R
            </Typography>
          </Box>
        )}
      </Box>

      {/* Primary navigation */}
      <List sx={{ px: sidebarOpen ? 1 : 0.5, py: 1.5 }}>
        {NAV_ITEMS.map((item) => {
          const active = pathname === item.path || pathname.startsWith(item.path + '/')
          return (
            <ListItem key={item.path} disablePadding sx={{ mb: 0.5 }}>
              <Tooltip title={!sidebarOpen ? item.label : ''} placement="right">
                <ListItemButton
                  selected={active}
                  onClick={() => navigate(item.path)}
                  sx={{
                    borderRadius: '10px',
                    justifyContent: sidebarOpen ? 'flex-start' : 'center',
                    px: sidebarOpen ? 1.75 : 1,
                    minHeight: 46,
                    position: 'relative',
                    transition:
                      'background-color 140ms ease, color 140ms ease, box-shadow 140ms ease',
                    '&.Mui-selected': {
                      bgcolor: 'rgba(129,140,248,0.14)',
                      boxShadow: '0 0 0 1px rgba(129,140,248,0.2) inset',
                      '&:hover': { bgcolor: 'rgba(129,140,248,0.2)' },
                      '&::before': {
                        content: '""',
                        position: 'absolute',
                        left: 4,
                        top: 10,
                        bottom: 10,
                        width: 3,
                        borderRadius: 2,
                        background: tokens.gradients.primary,
                        boxShadow: '0 0 10px rgba(129,140,248,0.6)',
                      },
                    },
                    '&:hover': {
                      bgcolor: 'rgba(129,140,248,0.07)',
                      boxShadow: '0 0 0 1px rgba(129,140,248,0.12) inset',
                      '& .MuiListItemIcon-root': { color: '#c4b5fd' },
                    },
                  }}
                >
                  <ListItemIcon
                    sx={{
                      minWidth: sidebarOpen ? 40 : 'auto',
                      color: active ? '#a78bfa' : 'text.secondary',
                      transition: 'color 120ms ease',
                      '& .MuiSvgIcon-root': { fontSize: '1.375rem' },
                    }}
                  >
                    {item.icon}
                  </ListItemIcon>
                  {sidebarOpen && (
                    <ListItemText
                      primary={item.label}
                      primaryTypographyProps={{
                        fontSize: '0.9375rem',
                        fontWeight: active ? 600 : 500,
                        color: active ? '#c4b5fd' : 'text.secondary',
                        letterSpacing: '-0.005em',
                        noWrap: true,
                      }}
                    />
                  )}
                </ListItemButton>
              </Tooltip>
            </ListItem>
          )
        })}
      </List>

      {/* Footer navigation (Profile, Settings) — pinned to the bottom of the drawer */}
      <List
        sx={{
          px: sidebarOpen ? 1 : 0.5,
          py: 1.5,
          mt: 'auto',
          borderTop: '1px solid rgba(255,255,255,0.05)',
        }}
      >
        {FOOTER_NAV_ITEMS.map((item) => {
          const active = pathname === item.path || pathname.startsWith(item.path + '/')
          return (
            <ListItem key={item.path} disablePadding sx={{ mb: 0.5 }}>
              <Tooltip title={!sidebarOpen ? item.label : ''} placement="right">
                <ListItemButton
                  selected={active}
                  onClick={() => navigate(item.path)}
                  sx={{
                    borderRadius: '8px',
                    justifyContent: sidebarOpen ? 'flex-start' : 'center',
                    px: sidebarOpen ? 1.5 : 1,
                    minHeight: 40,
                    position: 'relative',
                    transition: 'background-color 120ms ease, color 120ms ease',
                    '&.Mui-selected': {
                      bgcolor: 'rgba(129,140,248,0.12)',
                      '&:hover': { bgcolor: 'rgba(129,140,248,0.18)' },
                      '&::before': {
                        content: '""',
                        position: 'absolute',
                        left: 4,
                        top: 8,
                        bottom: 8,
                        width: 3,
                        borderRadius: 2,
                        background: tokens.gradients.primary,
                      },
                    },
                    '&:hover': {
                      bgcolor: 'rgba(255,255,255,0.04)',
                    },
                  }}
                >
                  <ListItemIcon
                    sx={{
                      minWidth: sidebarOpen ? 40 : 'auto',
                      color: active ? '#a78bfa' : 'text.secondary',
                      transition: 'color 120ms ease',
                      '& .MuiSvgIcon-root': { fontSize: '1.375rem' },
                    }}
                  >
                    {item.icon}
                  </ListItemIcon>
                  {sidebarOpen && (
                    <ListItemText
                      primary={item.label}
                      primaryTypographyProps={{
                        fontSize: '0.9375rem',
                        fontWeight: active ? 600 : 500,
                        color: active ? '#c4b5fd' : 'text.secondary',
                        letterSpacing: '-0.005em',
                        noWrap: true,
                      }}
                    />
                  )}
                </ListItemButton>
              </Tooltip>
            </ListItem>
          )
        })}
      </List>
    </Drawer>
  )
}
