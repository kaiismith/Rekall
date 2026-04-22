import AppBar from '@mui/material/AppBar'
import Toolbar from '@mui/material/Toolbar'
import IconButton from '@mui/material/IconButton'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import Avatar from '@mui/material/Avatar'
import Tooltip from '@mui/material/Tooltip'
import MenuIcon from '@mui/icons-material/Menu'
import MenuOpenIcon from '@mui/icons-material/MenuOpen'
import NotificationsNoneIcon from '@mui/icons-material/NotificationsNone'
import { SIDEBAR_WIDTH, SIDEBAR_COLLAPSED_WIDTH } from '@/constants'
import { useUIStore } from '@/store/uiStore'

export function TopBar() {
  const { sidebarOpen, toggleSidebar } = useUIStore()
  const drawerWidth = sidebarOpen ? SIDEBAR_WIDTH : SIDEBAR_COLLAPSED_WIDTH

  return (
    <AppBar
      position="fixed"
      sx={{
        width: { sm: `calc(100% - ${drawerWidth}px)` },
        ml: { sm: `${drawerWidth}px` },
        transition: 'width 0.2s ease, margin-left 0.2s ease',
        zIndex: (theme) => theme.zIndex.drawer - 1,
      }}
    >
      <Toolbar sx={{ gap: 1, minHeight: '56px !important' }}>
        <IconButton
          color="inherit"
          edge="start"
          onClick={toggleSidebar}
          aria-label="toggle sidebar"
          size="small"
        >
          {sidebarOpen ? <MenuOpenIcon /> : <MenuIcon />}
        </IconButton>

        {/* Spacer */}
        <Box flexGrow={1} />

        <Tooltip title="Notifications">
          <IconButton color="inherit" size="small">
            <NotificationsNoneIcon fontSize="small" />
          </IconButton>
        </Tooltip>

        <Tooltip title="Account">
          <Avatar
            sx={{
              width: 32,
              height: 32,
              bgcolor: 'primary.main',
              fontSize: '0.8rem',
              cursor: 'pointer',
            }}
          >
            U
          </Avatar>
        </Tooltip>

        <Typography variant="body2" color="text.secondary" sx={{ display: { xs: 'none', sm: 'block' } }}>
          User
        </Typography>
      </Toolbar>
    </AppBar>
  )
}
