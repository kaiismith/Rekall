import { useEffect, useRef, useState } from 'react'
import { Box, Chip, Stack, Typography } from '@mui/material'
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome'
import HelpOutlineIcon from '@mui/icons-material/HelpOutline'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

import type { KatHealthResponse, KatNoteDTO, KatStatus } from '@/types/kat'

/** useTypewriter reveals `target` one character at a time at the given
 *  cadence. Decouples visible typing speed from the actual stream rate so
 *  short fast responses still feel like Kat is "thinking aloud". When
 *  `target` grows, the cursor advances toward the new end at the same
 *  rate. When `target` shrinks (new note replaces old), the visible cursor
 *  resets and re-animates from scratch. */
function useTypewriter(target: string, charsPerSecond = 45): string {
  const [count, setCount] = useState(0)
  const targetRef = useRef(target)

  useEffect(() => {
    // Reset when target shrinks below the current cursor (new generation).
    if (target.length < targetRef.current.length) {
      setCount(0)
    }
    targetRef.current = target
  }, [target])

  useEffect(() => {
    if (count >= target.length) return
    const id = window.setTimeout(
      () => {
        setCount((c) => Math.min(target.length, c + 1))
      },
      1000 / Math.max(1, charsPerSecond),
    )
    return () => window.clearTimeout(id)
  }, [count, target.length, charsPerSecond])

  return target.slice(0, count)
}

/** Render Kat-produced text as markdown. Restricted set: only the inline
 *  formatting features the kat-v1 prompt is allowed to emit (bold, italic,
 *  inline code). Block-level constructs (headings, code fences, tables,
 *  images, links) are passed through as plain text — the prompt forbids
 *  them but if the model gets creative we won't blow up the layout. */
function Md({ text, variant = 'body2' }: { text: string; variant?: 'body1' | 'body2' }) {
  return (
    <Typography
      component="div"
      variant={variant}
      sx={{
        // react-markdown wraps content in a <p> by default; collapse that
        // so it matches the visual rhythm of plain Typography.
        '& p': { m: 0 },
        '& strong': { fontWeight: 600 },
        '& em': { fontStyle: 'italic' },
        '& code': {
          fontFamily: 'monospace',
          fontSize: '0.85em',
          px: 0.5,
          py: 0.1,
          borderRadius: 0.5,
          bgcolor: 'rgba(255,255,255,0.06)',
        },
      }}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        // Disallow block elements the prompt forbids; if the model emits
        // them anyway, render their text content without the block markup.
        disallowedElements={['h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'pre', 'table', 'img', 'a']}
        unwrapDisallowed
      >
        {text}
      </ReactMarkdown>
    </Typography>
  )
}

/** Strip the kat-v1 section markers and return just the SUMMARY body so
 *  the user sees clean prose during streaming, not "SUMMARY: ..."
 *  template noise. Tolerant of partial input (early chunks that haven't
 *  emitted SUMMARY: yet → returns ""). */
function extractStreamingSummary(raw: string): string {
  if (!raw) return ''
  const upper = raw.toUpperCase()
  const start = upper.indexOf('SUMMARY:')
  if (start < 0) return '' // model hasn't emitted the marker yet
  const after = raw.slice(start + 'SUMMARY:'.length)
  // Stop at the first subsequent section marker if it has appeared.
  const upperAfter = after.toUpperCase()
  let end = after.length
  for (const marker of ['KEY POINTS:', 'OPEN QUESTIONS:']) {
    const idx = upperAfter.indexOf(marker)
    if (idx >= 0 && idx < end) end = idx
  }
  return after.slice(0, end).trim()
}

interface KatPanelProps {
  status: KatStatus
  latestNote: KatNoteDTO | null
  health: KatHealthResponse | null
  /** Running raw text from an in-flight streaming response. Rendered in
   *  the summary slot while status==='streaming' so users see Kat
   *  type out the note token-by-token. */
  streamingPartial?: string | null
}

/** Kat live notes panel.
 *
 *  Renders one of five visual states:
 *    - `idle`        — bootstrap probe in flight; show a subtle skeleton
 *    - `warming_up`  — Foundry configured, no notes yet ("Kat is listening…")
 *    - `live`        — render the latest summary + bullets + open questions
 *    - `offline`     — Foundry not configured; greyed card with operator hint
 *    - `error`       — probe failed; show a small notice
 *
 *  The "Notes are not saved — they live only during this meeting" hint is
 *  visible in `live` and `warming_up` so the ephemerality is observable to
 *  the user without reading any docs.
 */
export function KatPanel({ status, latestNote, health, streamingPartial }: KatPanelProps) {
  // Drive the typewriter from the parsed-clean summary text so the user
  // sees ordinary prose appearing word-by-word, not raw "SUMMARY:" markers.
  const cleanPartial = extractStreamingSummary(streamingPartial ?? '')
  const typed = useTypewriter(cleanPartial, 45)
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Stack
        direction="row"
        alignItems="center"
        spacing={1}
        sx={{ pb: 1.5, borderBottom: '1px solid', borderColor: 'divider', flexShrink: 0 }}
      >
        <AutoAwesomeIcon fontSize="small" sx={{ color: 'primary.main' }} />
        <Typography variant="subtitle2" fontWeight={600} sx={{ flexGrow: 1 }}>
          Kat — live notes
        </Typography>
        {status === 'live' && <Chip label="Live" size="small" color="success" variant="outlined" />}
        {status === 'streaming' && (
          <Chip label="Generating…" size="small" color="primary" variant="outlined" />
        )}
        {status === 'warming_up' && <Chip label="Warming up" size="small" variant="outlined" />}
        {status === 'empty' && <Chip label="No speech" size="small" variant="outlined" />}
        {status === 'offline' && <Chip label="Offline" size="small" variant="outlined" />}
        {status === 'error' && (
          <Chip label="Error" size="small" color="warning" variant="outlined" />
        )}
      </Stack>

      <Box sx={{ flex: 1, overflowY: 'auto', mt: 2, pr: 1 }}>
        {renderBody({ status, latestNote, streamingPartial, typed })}
      </Box>

      {(status === 'live' ||
        status === 'warming_up' ||
        status === 'empty' ||
        status === 'streaming') && (
        <Stack
          spacing={0.25}
          sx={{
            pt: 1.5,
            mt: 1,
            borderTop: '1px solid',
            borderColor: 'divider',
            flexShrink: 0,
          }}
        >
          <Typography variant="caption" color="text.secondary">
            Notes are not saved — they live only during this meeting.
          </Typography>
          <Typography variant="caption" color="text.disabled">
            Powered by Kat
            {health?.provider ? ` • ${formatProvider(health.provider)}` : ''}
            {health?.deployment ? ` • ${health.deployment}` : ''}
          </Typography>
        </Stack>
      )}
    </Box>
  )
}

function renderBody({
  status,
  latestNote,
  streamingPartial,
  typed,
}: Pick<KatPanelProps, 'status' | 'latestNote' | 'streamingPartial'> & { typed: string }) {
  // Streaming: render the typewriter-paced summary text with a blinking
  // caret so users see Kat "thinking aloud". We hide the raw section
  // markers (SUMMARY: / KEY POINTS: / OPEN QUESTIONS:) because they look
  // like prompt template leak — the final structured view replaces this
  // panel the moment the stream completes.
  if (status === 'streaming' && streamingPartial) {
    return (
      <Box>
        <Typography variant="caption" color="text.disabled" sx={{ display: 'block', mb: 0.5 }}>
          Kat is generating…
        </Typography>
        <Typography
          variant="body1"
          sx={{
            whiteSpace: 'pre-wrap',
            // Blinking caret while typing.
            '&::after': {
              content: '"▍"',
              opacity: 0.6,
              ml: 0.25,
              animation: 'katBlink 1s steps(2, start) infinite',
            },
            '@keyframes katBlink': {
              '50%': { opacity: 0 },
            },
          }}
          data-testid="kat-streaming"
        >
          {typed || 'Thinking…'}
        </Typography>
      </Box>
    )
  }

  if (status === 'offline') {
    return (
      <Box>
        <Typography variant="body2" color="text.secondary">
          Kat is offline.
        </Typography>
        <Typography variant="caption" color="text.disabled">
          Ask your administrator to configure an AI provider (Azure AI Foundry or OpenAI).
        </Typography>
      </Box>
    )
  }

  if (status === 'error') {
    return (
      <Typography variant="body2" color="warning.main">
        Couldn&apos;t reach Kat. The captions UX is unaffected.
      </Typography>
    )
  }

  if (status === 'empty' && !latestNote) {
    return (
      <Box>
        <Typography variant="body2" color="text.secondary">
          There&apos;s nothing to take notes.
        </Typography>
        <Typography variant="caption" color="text.disabled">
          Notes will appear once someone in the meeting starts talking.
        </Typography>
      </Box>
    )
  }

  if (status === 'idle' || (status === 'warming_up' && !latestNote)) {
    return (
      <Typography variant="body2" color="text.secondary">
        Kat is listening… first notes in a moment.
      </Typography>
    )
  }

  if (!latestNote) {
    return null
  }

  return (
    <Stack spacing={2} data-testid="kat-note">
      <Box>
        <Typography variant="caption" color="text.disabled" sx={{ display: 'block', mb: 0.5 }}>
          Updated {formatTimestamp(latestNote.window_ended_at)}
          {' · '}covers last {formatDuration(latestNote)}
        </Typography>
        <Box data-testid="kat-summary">
          <Md text={latestNote.summary} variant="body1" />
        </Box>
      </Box>

      {latestNote.key_points.length > 0 && (
        <Box>
          <Typography variant="overline" color="text.secondary">
            Key points
          </Typography>
          <Box
            component="ul"
            sx={{
              pl: 3,
              m: 0,
              // MUI's CSS reset clears list-style on ul. Restore the disc
              // marker explicitly so each entry actually renders as a bullet.
              listStyleType: 'disc',
              '& li': {
                display: 'list-item',
                mb: 0.25,
              },
            }}
            data-testid="kat-key-points"
          >
            {latestNote.key_points.map((kp, i) => (
              <Box component="li" key={i}>
                <Md text={kp} />
              </Box>
            ))}
          </Box>
        </Box>
      )}

      {latestNote.open_questions.length > 0 && (
        <Box>
          <Typography variant="overline" color="text.secondary">
            Open questions
          </Typography>
          <Stack spacing={0.5} data-testid="kat-open-questions">
            {latestNote.open_questions.map((q, i) => (
              <Stack direction="row" spacing={1} key={i} alignItems="flex-start">
                <HelpOutlineIcon fontSize="small" color="info" />
                <Box sx={{ fontStyle: 'italic' }}>
                  <Md text={q} />
                </Box>
              </Stack>
            ))}
          </Stack>
        </Box>
      )}
    </Stack>
  )
}

function formatProvider(provider: string): string {
  switch (provider) {
    case 'foundry':
      return 'Azure AI Foundry'
    case 'openai':
      return 'OpenAI'
    default:
      return provider
  }
}

function formatTimestamp(iso: string): string {
  try {
    const d = new Date(iso)
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    return iso
  }
}

function formatDuration(note: KatNoteDTO): string {
  try {
    const start = new Date(note.window_started_at).getTime()
    const end = new Date(note.window_ended_at).getTime()
    const sec = Math.max(0, Math.round((end - start) / 1000))
    if (sec < 60) return `${sec}s`
    const min = Math.round(sec / 60)
    return `${min} min`
  } catch {
    return '—'
  }
}
