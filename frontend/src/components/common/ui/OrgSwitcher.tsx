import { useEffect, useState } from 'react'
import {
  Button,
  Divider,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Skeleton,
  Typography,
} from '@mui/material'
import BusinessIcon from '@mui/icons-material/Business'
import PersonOutlineIcon from '@mui/icons-material/PersonOutline'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import { matchPath, useLocation, useNavigate, useParams } from 'react-router-dom'
import { ROUTES, buildScopedRoute } from '@/constants'
import { useOrgsStore } from '@/store/orgsStore'

/**
 * Top-bar dropdown that lets the user see which org they're "in" and jump
 * between orgs (or back to Personal). Width auto-sizes to the current label.
 *
 * URL is the source of truth — selecting an entry navigates; the trigger
 * label is derived from the current pathname plus the params, never from
 * a "current org" store.
 */
export function OrgSwitcher() {
  const orgs = useOrgsStore((s) => s.orgs)
  const isLoading = useOrgsStore((s) => s.isLoading)
  const load = useOrgsStore((s) => s.load)
  const getOrgName = useOrgsStore((s) => s.getOrgName)

  const navigate = useNavigate()
  const { pathname } = useLocation()
  const params = useParams<{ id?: string; orgId?: string }>()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const open = Boolean(anchor)

  useEffect(() => {
    if (orgs === null && !isLoading) void load()
  }, [orgs, isLoading, load])

  const onScoped = matchPath('/organizations/:id/*', pathname) !== null
  const onOrgPage = matchPath('/organizations/:id', pathname) !== null
  const currentOrgId = params.orgId ?? params.id
  const currentOrgName =
    (onScoped || onOrgPage) && currentOrgId ? getOrgName(currentOrgId) : undefined
  const triggerLabel = currentOrgName ?? 'Personal'

  const close = () => setAnchor(null)
  const goto = (path: string) => {
    close()
    navigate(path)
  }

  return (
    <>
      <Button
        onClick={(e) => setAnchor(e.currentTarget)}
        endIcon={<ExpandMoreIcon />}
        aria-haspopup="menu"
        aria-expanded={open || undefined}
        size="small"
        sx={{
          color: 'text.primary',
          textTransform: 'none',
          fontWeight: 600,
          maxWidth: { xs: 140, sm: 220 },
          '& .MuiButton-endIcon': { ml: 0.5 },
          '& .MuiButton-startIcon, & .MuiTypography-root': {
            display: { xs: 'none', sm: 'inline-flex' },
          },
        }}
      >
        {orgs === null ? (
          <Skeleton width={80} height={20} />
        ) : (
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {triggerLabel}
          </span>
        )}
      </Button>

      <Menu
        anchorEl={anchor}
        open={open}
        onClose={close}
        slotProps={{ paper: { sx: { minWidth: 260 } } }}
      >
        <MenuItem onClick={() => goto(ROUTES.DASHBOARD)}>
          <ListItemIcon>
            <PersonOutlineIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText primary="Personal" secondary="Open meetings and calls" />
        </MenuItem>

        <Divider />

        {orgs && orgs.length > 0 ? (
          orgs.map((o) => (
            <MenuItem key={o.id} onClick={() => goto(buildScopedRoute.org(o.id))}>
              <ListItemIcon>
                <BusinessIcon fontSize="small" />
              </ListItemIcon>
              <ListItemText
                primary={o.name}
                secondary={o.slug}
                primaryTypographyProps={{ noWrap: true }}
                secondaryTypographyProps={{ noWrap: true }}
              />
            </MenuItem>
          ))
        ) : (
          <MenuItem disabled>
            <Typography variant="body2" color="text.secondary">
              You aren&apos;t in any organizations yet.
            </Typography>
          </MenuItem>
        )}

        <Divider />

        <MenuItem onClick={() => goto(ROUTES.ORGANIZATIONS)}>
          <ListItemText primary="Manage organizations" />
        </MenuItem>
      </Menu>
    </>
  )
}
