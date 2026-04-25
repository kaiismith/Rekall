import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AppBar from '@mui/material/AppBar'
import Toolbar from '@mui/material/Toolbar'
import IconButton from '@mui/material/IconButton'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import Avatar from '@mui/material/Avatar'
import Tooltip from '@mui/material/Tooltip'
import Menu from '@mui/material/Menu'
import MenuItem from '@mui/material/MenuItem'
import Divider from '@mui/material/Divider'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import MenuIcon from '@mui/icons-material/Menu'
import MenuOpenIcon from '@mui/icons-material/MenuOpen'
import NotificationsNoneIcon from '@mui/icons-material/NotificationsNone'
import HelpOutlineOutlinedIcon from '@mui/icons-material/HelpOutlineOutlined'
import LogoutOutlinedIcon from '@mui/icons-material/LogoutOutlined'
import { SIDEBAR_WIDTH, SIDEBAR_COLLAPSED_WIDTH, ROUTES } from '@/constants'
import { useUIStore } from '@/store/uiStore'
import { useAuthStore } from '@/store/authStore'
import { authService } from '@/services/authService'
import { tokens } from '@/theme'
import { OrgSwitcher } from '@/components/common/ui'

function initialsFromName(name?: string): string {
  if (!name) return 'U'
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return 'U'
  const first = parts[0]!.charAt(0)
  const last = parts.length > 1 ? parts[parts.length - 1]!.charAt(0) : ''
  return (first + last).toUpperCase()
}

export function TopBar() {
  const navigate = useNavigate()
  const { sidebarOpen, toggleSidebar } = useUIStore()
  const { user, clearAuth } = useAuthStore()
  const drawerWidth = sidebarOpen ? SIDEBAR_WIDTH : SIDEBAR_COLLAPSED_WIDTH

  const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null)
  const menuOpen = Boolean(menuAnchor)

  const initials = initialsFromName(user?.full_name)
  const displayName = user?.full_name ?? 'User'
  const email = user?.email ?? ''

  const closeMenu = () => setMenuAnchor(null)

  const handleSignOut = async () => {
    closeMenu()
    try {
      await authService.logout()
    } catch {
      // Even if the backend call fails, clear local state so the user is out.
    }
    clearAuth()
    navigate(ROUTES.LOGIN, { replace: true })
  }

  return (
    <AppBar
      position="fixed"
      sx={{
        width: { sm: `calc(100% - ${drawerWidth}px)` },
        ml: { sm: `${drawerWidth}px` },
        transition: 'width 0.2s ease, margin-left 0.2s ease',
        zIndex: (theme) => theme.zIndex.drawer - 1,
        backgroundColor: 'rgba(13,15,20,0.7)',
        backdropFilter: 'blur(12px)',
        borderBottom: '1px solid rgba(255,255,255,0.06)',
      }}
    >
      <Toolbar
        disableGutters
        sx={{
          gap: 1,
          minHeight: '56px !important',
          // disableGutters drops the default 24px Toolbar padding; we apply
          // tighter custom horizontal padding so the hamburger sits close
          // to the sidebar edge instead of floating in empty space.
          px: { xs: 1.5, sm: 2 },
        }}
      >
        <IconButton
          color="inherit"
          edge="start"
          onClick={toggleSidebar}
          aria-label="toggle sidebar"
          size="small"
          sx={{ color: 'text.secondary' }}
        >
          {sidebarOpen ? <MenuOpenIcon /> : <MenuIcon />}
        </IconButton>

        {/* Spacer */}
        <Box flexGrow={1} />

        <OrgSwitcher />

        <Tooltip title="Notifications">
          <IconButton size="small" sx={{ color: 'text.secondary' }}>
            <NotificationsNoneIcon fontSize="small" />
          </IconButton>
        </Tooltip>

        <Tooltip title="Help & docs">
          <IconButton
            size="small"
            sx={{ color: 'text.secondary' }}
            onClick={() => navigate(ROUTES.HELP)}
            aria-label="Help and documentation"
          >
            <HelpOutlineOutlinedIcon fontSize="small" />
          </IconButton>
        </Tooltip>

        <Tooltip title="Account">
          <IconButton
            onClick={(e) => setMenuAnchor(e.currentTarget)}
            size="small"
            aria-controls={menuOpen ? 'account-menu' : undefined}
            aria-haspopup="true"
            aria-expanded={menuOpen ? 'true' : undefined}
            sx={{ ml: 0.5, p: 0.25 }}
          >
            <Avatar
              sx={{
                width: 32,
                height: 32,
                fontSize: '0.8rem',
                fontWeight: 700,
                letterSpacing: '-0.02em',
                color: '#0a0b12',
                background: tokens.gradients.primary,
                boxShadow: '0 2px 8px rgba(129,140,248,0.35)',
              }}
            >
              {initials}
            </Avatar>
          </IconButton>
        </Tooltip>

        <Menu
          id="account-menu"
          anchorEl={menuAnchor}
          open={menuOpen}
          onClose={closeMenu}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
          transformOrigin={{ vertical: 'top', horizontal: 'right' }}
          slotProps={{
            paper: {
              sx: {
                mt: 1,
                minWidth: 240,
                borderRadius: '12px',
                border: '1px solid rgba(255,255,255,0.06)',
                bgcolor: '#151823',
                boxShadow: '0 12px 40px rgba(0,0,0,0.5)',
                overflow: 'hidden',
              },
            },
          }}
        >
          {/* Identity block */}
          <Box sx={{ px: 2, py: 1.5 }}>
            <Typography
              variant="body2"
              sx={{ fontWeight: 600, color: 'text.primary', letterSpacing: '-0.005em' }}
              noWrap
            >
              {displayName}
            </Typography>
            {email && (
              <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                {email}
              </Typography>
            )}
          </Box>

          <Divider sx={{ borderColor: 'rgba(255,255,255,0.06)' }} />

          <MenuItem
            onClick={handleSignOut}
            sx={{
              color: '#fca5a5',
              '&:hover': { bgcolor: 'rgba(239,68,68,0.08)' },
              '& .MuiListItemIcon-root': { color: '#fca5a5' },
            }}
          >
            <ListItemIcon>
              <LogoutOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText
              primary="Sign out"
              primaryTypographyProps={{ variant: 'body2', fontWeight: 500 }}
            />
          </MenuItem>
        </Menu>
      </Toolbar>
    </AppBar>
  )
}
