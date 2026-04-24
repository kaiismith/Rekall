import Box from '@mui/material/Box'
import Toolbar from '@mui/material/Toolbar'
import { Outlet } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'

/**
 * Root application shell: fixed TopBar + collapsible Sidebar + scrollable content area.
 *
 * The Sidebar is a permanent MUI Drawer — its outer wrapper occupies its
 * width inside the flex container, so `<main>` only needs `flexGrow: 1` to
 * take the remaining space. We deliberately do NOT add `ml: drawerWidth` /
 * `width: calc(100% - drawerWidth)` here; that would double-account for the
 * sidebar and leave a wide empty gap between the sidebar and the content.
 */
export function Layout() {
  return (
    <Box sx={{ display: 'flex', minHeight: '100vh', bgcolor: 'background.default' }}>
      <TopBar />
      <Sidebar />

      <Box
        component="main"
        sx={{
          flexGrow: 1,
          minWidth: 0,
          minHeight: '100vh',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {/* Spacer to push content below the AppBar */}
        <Toolbar sx={{ minHeight: '56px !important' }} />

        {/* Comfortable gutter against the sidebar — enough room to breathe
            without re-introducing the floating-content feel. */}
        <Box
          sx={{
            flex: 1,
            px: { xs: 2, sm: 6 },
            py: { xs: 2, sm: 3 },
            width: '100%',
            maxWidth: 'none',
          }}
        >
          <Outlet />
        </Box>
      </Box>
    </Box>
  )
}
