import { useEffect, useState } from 'react'
import {
  Box,
  Chip,
  Collapse,
  Divider,
  IconButton,
  List,
  ListItemButton,
  ListItemText,
  Popover,
  Skeleton,
  Typography,
} from '@mui/material'
import ExpandLessIcon from '@mui/icons-material/ExpandLess'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import PublicOutlinedIcon from '@mui/icons-material/PublicOutlined'
import LayersOutlinedIcon from '@mui/icons-material/LayersOutlined'
import type { Scope } from '@/types/scope'
import { useOrgsStore } from '@/store/orgsStore'
import { useDeptsStore } from '@/store/deptsStore'

interface ScopePickerProps {
  /** Currently active scope filter, or null for "All scopes". */
  value: Scope | null
  onChange: (scope: Scope | null) => void
}

/**
 * Header-mounted filter chip (used on the flat Meetings & Calls pages).
 * Clicking opens a popover with: All / Personal / one expandable row per org
 * → its departments. Selecting a row writes through `onChange`; the URL is
 * the source of truth so reload restores the filter.
 */
export function ScopePicker({ value, onChange }: ScopePickerProps) {
  const orgs = useOrgsStore((s) => s.orgs)
  const loadOrgs = useOrgsStore((s) => s.load)
  const getOrgName = useOrgsStore((s) => s.getOrgName)
  const ensureDepts = useDeptsStore((s) => s.ensureLoaded)
  const listDepts = useDeptsStore((s) => s.listForOrg)
  const getDeptName = useDeptsStore((s) => s.getDeptName)

  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})
  const open = Boolean(anchor)

  useEffect(() => {
    if (orgs === null) void loadOrgs()
  }, [orgs, loadOrgs])

  const triggerLabel = (() => {
    if (!value) return 'All scopes'
    if (value.type === 'open') return 'Personal (open)'
    if (value.type === 'organization') return getOrgName(value.id) ?? 'Organization'
    const orgName = getOrgName(value.orgId) ?? 'Org'
    const deptName = getDeptName(value.orgId, value.id) ?? 'Dept'
    return `${orgName} › ${deptName}`
  })()

  const close = () => setAnchor(null)
  const select = (s: Scope | null) => {
    close()
    onChange(s)
  }

  const toggle = (orgId: string) => {
    setExpanded((s) => {
      const next = !s[orgId]
      if (next) void ensureDepts(orgId)
      return { ...s, [orgId]: next }
    })
  }

  return (
    <>
      <Chip
        label={triggerLabel}
        onClick={(e) => setAnchor(e.currentTarget)}
        size="small"
        deleteIcon={value ? undefined : <ExpandMoreIcon fontSize="small" />}
        onDelete={value ? () => onChange(null) : undefined}
        sx={{
          cursor: 'pointer',
          maxWidth: 240,
          fontWeight: 500,
          '& .MuiChip-label': { overflow: 'hidden', textOverflow: 'ellipsis' },
        }}
      />

      <Popover
        anchorEl={anchor}
        open={open}
        onClose={close}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        slotProps={{ paper: { sx: { minWidth: 280, maxHeight: 420, mt: 1 } } }}
      >
        <Typography
          variant="overline"
          sx={{ display: 'block', px: 2, pt: 1.5, color: 'text.secondary' }}
        >
          Filter by scope
        </Typography>
        <List dense>
          <ListItemButton selected={value === null} onClick={() => select(null)}>
            <ListItemText primary="All scopes" />
          </ListItemButton>
          <ListItemButton
            selected={value?.type === 'open'}
            onClick={() => select({ type: 'open' })}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25 }}>
              <PublicOutlinedIcon fontSize="small" sx={{ color: 'text.secondary' }} />
              <ListItemText primary="Personal (open)" />
            </Box>
          </ListItemButton>
        </List>

        {orgs === null ? (
          <Box sx={{ px: 2, py: 1 }}>
            <Skeleton width="80%" />
          </Box>
        ) : orgs.length === 0 ? null : (
          <>
            <Divider />
            <List dense disablePadding>
              {orgs.map((o) => {
                const exp = !!expanded[o.id]
                const depts = listDepts(o.id)
                const orgSelected = value?.type === 'organization' && value.id === o.id
                return (
                  <Box key={o.id}>
                    <ListItemButton
                      selected={orgSelected}
                      onClick={() => select({ type: 'organization', id: o.id })}
                    >
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, flex: 1, minWidth: 0 }}>
                        <LayersOutlinedIcon fontSize="small" sx={{ color: 'text.secondary' }} />
                        <ListItemText primary={o.name} primaryTypographyProps={{ noWrap: true }} />
                      </Box>
                      <IconButton
                        size="small"
                        onClick={(e) => {
                          e.stopPropagation()
                          toggle(o.id)
                        }}
                        aria-label={exp ? 'Collapse' : 'Expand'}
                      >
                        {exp ? (
                          <ExpandLessIcon fontSize="small" />
                        ) : (
                          <ExpandMoreIcon fontSize="small" />
                        )}
                      </IconButton>
                    </ListItemButton>
                    <Collapse in={exp} unmountOnExit>
                      {depts === undefined ? (
                        <Box sx={{ pl: 6, py: 1 }}>
                          <Skeleton width="60%" />
                        </Box>
                      ) : depts.length === 0 ? (
                        <Box sx={{ pl: 6, py: 1 }}>
                          <Typography variant="caption" color="text.secondary">
                            No departments
                          </Typography>
                        </Box>
                      ) : (
                        depts.map((d) => {
                          const deptSelected =
                            value?.type === 'department' &&
                            value.id === d.id &&
                            value.orgId === o.id
                          return (
                            <ListItemButton
                              key={d.id}
                              selected={deptSelected}
                              onClick={() => select({ type: 'department', id: d.id, orgId: o.id })}
                              sx={{ pl: 6 }}
                            >
                              <ListItemText
                                primary={d.name}
                                primaryTypographyProps={{ noWrap: true, fontSize: '0.875rem' }}
                              />
                            </ListItemButton>
                          )
                        })
                      )}
                    </Collapse>
                  </Box>
                )
              })}
            </List>
          </>
        )}
      </Popover>
    </>
  )
}
