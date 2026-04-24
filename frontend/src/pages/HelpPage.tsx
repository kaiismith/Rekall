import { useState } from 'react'
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Card,
  CardContent,
  Link,
  Stack,
  Typography,
} from '@mui/material'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import VideocamOutlinedIcon from '@mui/icons-material/VideocamOutlined'
import PhoneInTalkOutlinedIcon from '@mui/icons-material/PhoneInTalkOutlined'
import BusinessOutlinedIcon from '@mui/icons-material/BusinessOutlined'
import KeyboardOutlinedIcon from '@mui/icons-material/KeyboardOutlined'
import ShieldOutlinedIcon from '@mui/icons-material/ShieldOutlined'
import OpenInNewRoundedIcon from '@mui/icons-material/OpenInNewRounded'
import { Link as RouterLink } from 'react-router-dom'
import { PageHeader, KeyChip } from '@/components/common/ui'
import { ROUTES } from '@/constants'

interface Section {
  id: string
  icon: React.ReactNode
  title: string
  blurb: string
  items: { q: string; a: React.ReactNode }[]
}

const SECTIONS: Section[] = [
  {
    id: 'meetings',
    icon: <VideocamOutlinedIcon />,
    title: 'Meetings',
    blurb: 'Creating, joining, and running meeting rooms.',
    items: [
      {
        q: 'How do I start a meeting?',
        a: (
          <>
            Open <Link component={RouterLink} to={ROUTES.NEW_MEETING}>New Meeting</Link> and
            press <strong>Create meeting</strong>, or use the keyboard shortcut{' '}
            <KeyChip>Ctrl/⌘</KeyChip> <KeyChip>Shift</KeyChip> <KeyChip>C</KeyChip>. A shareable
            meeting code is generated and you land directly in the room.
          </>
        ),
      },
      {
        q: 'How do I join with a code?',
        a: (
          <>
            From the same page, paste the meeting code into the "Join a meeting" input and
            press <strong>Join</strong>. If the meeting is private and you are outside the
            invited scope, you enter a waiting room until a host admits you.
          </>
        ),
      },
      {
        q: 'Who can see my screen when I share?',
        a: 'Everyone currently in the room. Screen sharing stops automatically when you close the tab the share originated from or when you click the stop-sharing control.',
      },
      {
        q: 'What are the participant limits?',
        a: 'Up to 50 participants per meeting. Past that the room refuses additional joins with a clear error.',
      },
    ],
  },
  {
    id: 'calls',
    icon: <PhoneInTalkOutlinedIcon />,
    title: 'Calls',
    blurb: 'How recorded calls flow into the platform and get processed.',
    items: [
      {
        q: 'How do calls get into Rekall?',
        a: 'Calls are ingested by the pipeline and appear on the Calls page automatically. No manual import step is required today.',
      },
      {
        q: 'Why is my call still "Processing"?',
        a: 'Transcription and enrichment can take a few minutes per call. The status updates in place — refresh the Calls page or open the call to see progress.',
      },
    ],
  },
  {
    id: 'orgs',
    icon: <BusinessOutlinedIcon />,
    title: 'Organizations',
    blurb: 'Workspaces, members, and departments.',
    items: [
      {
        q: 'Who can invite new members?',
        a: 'Owners and admins of the organization. Members can view the org but cannot invite or remove people.',
      },
      {
        q: 'What does "Private" mean on a meeting?',
        a: 'Only people inside the meeting\'s scope (organization or department) can join directly. Everyone else has to knock and be admitted by a host.',
      },
      {
        q: 'How do I delete an organization?',
        a: 'On the organization\'s detail page, scroll to the "Danger zone" and press Delete. You\'ll be asked to type the organization\'s slug to confirm — names can collide across orgs but slugs cannot.',
      },
    ],
  },
  {
    id: 'shortcuts',
    icon: <KeyboardOutlinedIcon />,
    title: 'Keyboard shortcuts',
    blurb: 'Power-user bindings. Toggle them off in Settings if they get in the way.',
    items: [
      {
        q: 'Ctrl/⌘ + Shift + C',
        a: (
          <>
            From the New Meeting page, instantly creates a meeting and drops you into the
            room — no mouse required. Disabled while an input is focused.
          </>
        ),
      },
      {
        q: 'How do I disable shortcuts?',
        a: (
          <>
            Open <Link component={RouterLink} to={ROUTES.SETTINGS}>Settings</Link> and turn
            off "Enable keyboard shortcuts". The setting persists to this browser only.
          </>
        ),
      },
    ],
  },
  {
    id: 'security',
    icon: <ShieldOutlinedIcon />,
    title: 'Security & privacy',
    blurb: 'How sessions, passwords, and tokens are handled.',
    items: [
      {
        q: 'Where is my password stored?',
        a: 'Only as a bcrypt hash on the server. Plaintext passwords are never logged or sent over the wire beyond the initial request.',
      },
      {
        q: 'What happens when I change my password?',
        a: 'Every other signed-in device is signed out immediately. The tab where you made the change stays signed in with a freshly rotated refresh token.',
      },
      {
        q: 'Where does my auth token live in the browser?',
        a: 'The access token lives in memory only — not in localStorage or sessionStorage. The refresh token is a HttpOnly cookie that JavaScript cannot read.',
      },
    ],
  },
]

function SectionCard({ section }: { section: Section }) {
  const [expanded, setExpanded] = useState<string | false>(false)

  return (
    <Card>
      <CardContent sx={{ p: { xs: 2.5, sm: 3 } }}>
        <Stack direction="row" spacing={1.75} alignItems="center" sx={{ mb: 1.5 }}>
          <Box
            sx={{
              width: 36,
              height: 36,
              borderRadius: '10px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              bgcolor: 'rgba(129,140,248,0.12)',
              color: '#a78bfa',
              flexShrink: 0,
            }}
          >
            {section.icon}
          </Box>
          <Box sx={{ minWidth: 0 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, letterSpacing: '-0.005em' }}>
              {section.title}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {section.blurb}
            </Typography>
          </Box>
        </Stack>

        <Box>
          {section.items.map((item, i) => {
            const panelId = `${section.id}-${i}`
            return (
              <Accordion
                key={panelId}
                expanded={expanded === panelId}
                onChange={(_, isOpen) => setExpanded(isOpen ? panelId : false)}
                disableGutters
                elevation={0}
                sx={{
                  bgcolor: 'transparent',
                  borderBottom: '1px solid rgba(255,255,255,0.04)',
                  '&:last-of-type': { borderBottom: 0 },
                  '&::before': { display: 'none' },
                }}
              >
                <AccordionSummary
                  expandIcon={<ExpandMoreIcon sx={{ color: 'text.secondary' }} />}
                  sx={{
                    px: 0,
                    '& .MuiAccordionSummary-content': { my: 1.5 },
                  }}
                >
                  <Typography variant="body2" sx={{ fontWeight: 500, color: 'text.primary' }}>
                    {item.q}
                  </Typography>
                </AccordionSummary>
                <AccordionDetails sx={{ px: 0, pt: 0, pb: 2 }}>
                  <Typography variant="body2" color="text.secondary" sx={{ lineHeight: 1.7 }}>
                    {item.a}
                  </Typography>
                </AccordionDetails>
              </Accordion>
            )
          })}
        </Box>
      </CardContent>
    </Card>
  )
}

export function HelpPage() {
  return (
    <Box>
      <PageHeader
        title="Help & documentation"
        subtitle="Frequently asked questions, shortcuts, and how Rekall handles your data."
      />

      <Stack spacing={2}>
        {SECTIONS.map((section) => (
          <SectionCard key={section.id} section={section} />
        ))}

        <Card>
          <CardContent sx={{ p: 3, display: 'flex', alignItems: 'center', gap: 2 }}>
            <OpenInNewRoundedIcon sx={{ color: '#a78bfa' }} />
            <Box sx={{ flex: 1 }}>
              <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                Didn&apos;t find what you need?
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Full API documentation is available at{' '}
                <Link href="/docs" target="_blank" rel="noopener noreferrer">
                  /docs
                </Link>{' '}
                when Swagger is enabled.
              </Typography>
            </Box>
          </CardContent>
        </Card>
      </Stack>
    </Box>
  )
}
