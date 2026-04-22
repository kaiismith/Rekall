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
import BusinessIcon from '@mui/icons-material/Business'
import VideoCallIcon from '@mui/icons-material/VideoCall'
import { useNavigate, useLocation } from 'react-router-dom'
import { SIDEBAR_WIDTH, SIDEBAR_COLLAPSED_WIDTH, ROUTES } from '@/constants'
import { useUIStore } from '@/store/uiStore'

const NAV_ITEMS = [
  { label: 'Dashboard', path: ROUTES.DASHBOARD, icon: <DashboardIcon fontSize="small" /> },
  { label: 'Calls', path: ROUTES.CALLS, icon: <PhoneInTalkIcon fontSize="small" /> },
  { label: 'Meetings', path: ROUTES.MEETINGS, icon: <VideoCallIcon fontSize="small" /> },
  { label: 'Organizations', path: ROUTES.ORGANIZATIONS, icon: <BusinessIcon fontSize="small" /> },
]

export function Sidebar() {
  const { sidebarOpen } = useUIStore()
  const navigate = useNavigate()
  const { pathname } = useLocation()

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
          py: 1.5,
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          minHeight: 56,
          borderBottom: '1px solid',
          borderColor: 'divider',
        }}
      >
        <Box
          sx={{
            width: 28,
            height: 28,
            borderRadius: '6px',
            bgcolor: 'primary.main',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexShrink: 0,
          }}
        >
          <Typography variant="caption" fontWeight={700} color="white" fontSize="0.7rem">
            R
          </Typography>
        </Box>
        {sidebarOpen && (
          <Typography variant="subtitle1" fontWeight={700} color="text.primary" noWrap>
            Rekall
          </Typography>
        )}
      </Box>

      {/* Navigation */}
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
                    borderRadius: '6px',
                    justifyContent: sidebarOpen ? 'flex-start' : 'center',
                    px: sidebarOpen ? 1.5 : 1,
                    minHeight: 40,
                  }}
                >
                  <ListItemIcon
                    sx={{
                      minWidth: sidebarOpen ? 36 : 'auto',
                      color: active ? 'primary.main' : 'text.secondary',
                    }}
                  >
                    {item.icon}
                  </ListItemIcon>
                  {sidebarOpen && (
                    <ListItemText
                      primary={item.label}
                      primaryTypographyProps={{
                        variant: 'body2',
                        fontWeight: active ? 600 : 400,
                        color: active ? 'text.primary' : 'text.secondary',
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
