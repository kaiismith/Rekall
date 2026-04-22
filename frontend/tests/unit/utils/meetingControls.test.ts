import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { EMOJI_LIST, PRESET_IMAGES, BACKGROUND_OPTIONS } from '@/config/meetingControls'

// ── EMOJI_LIST ────────────────────────────────────────────────────────────────

describe('EMOJI_LIST', () => {
  it('contains exactly 9 emojis', () => {
    expect(EMOJI_LIST).toHaveLength(9)
  })

  it('includes the expected set', () => {
    const expected = ['❤️', '👍', '🎉', '👏', '😂', '😮', '😢', '🤔', '👎']
    expected.forEach((e) => expect(EMOJI_LIST).toContain(e))
  })

  it('has no duplicates', () => {
    expect(new Set(EMOJI_LIST).size).toBe(EMOJI_LIST.length)
  })
})

// ── PRESET_IMAGES ─────────────────────────────────────────────────────────────

describe('PRESET_IMAGES', () => {
  it('contains exactly 5 presets', () => {
    expect(PRESET_IMAGES).toHaveLength(5)
  })

  it('each preset has a src and label', () => {
    PRESET_IMAGES.forEach(({ src, label }) => {
      expect(src).toBeTruthy()
      expect(label).toBeTruthy()
    })
  })

  it('all src paths start with /backgrounds/', () => {
    PRESET_IMAGES.forEach(({ src }) => {
      expect(src).toMatch(/^\/backgrounds\//)
    })
  })

  it('has no duplicate src paths', () => {
    const srcs = PRESET_IMAGES.map((p) => p.src)
    expect(new Set(srcs).size).toBe(srcs.length)
  })
})

// ── BACKGROUND_OPTIONS ────────────────────────────────────────────────────────

describe('BACKGROUND_OPTIONS', () => {
  it('starts with none', () => {
    expect(BACKGROUND_OPTIONS[0]).toEqual({ type: 'none' })
  })

  it('includes light and heavy blur', () => {
    expect(BACKGROUND_OPTIONS).toContainEqual({ type: 'blur', level: 'light' })
    expect(BACKGROUND_OPTIONS).toContainEqual({ type: 'blur', level: 'heavy' })
  })

  it('includes all 5 preset images', () => {
    const imageSrcs = BACKGROUND_OPTIONS
      .filter((o) => o.type === 'image')
      .map((o) => (o as { src: string }).src)
    expect(imageSrcs).toHaveLength(5)
    PRESET_IMAGES.forEach(({ src }) => expect(imageSrcs).toContain(src))
  })

  it('total count is 8 (1 none + 2 blur + 5 images)', () => {
    expect(BACKGROUND_OPTIONS).toHaveLength(8)
  })
})

// ── Custom background localStorage helpers ────────────────────────────────────

const BG_STORAGE_KEY = 'rekall_bg_preference'
const CUSTOM_BG_KEY = 'rekall_bg_custom'
const CUSTOM_SRC_SENTINEL = '__custom__'

describe('custom background localStorage contract', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    localStorage.clear()
  })

  it('stores a sentinel in preference key when custom image is active', () => {
    const dataUrl = 'data:image/png;base64,abc123'
    localStorage.setItem(CUSTOM_BG_KEY, dataUrl)
    localStorage.setItem(
      BG_STORAGE_KEY,
      JSON.stringify({ type: 'image', src: CUSTOM_SRC_SENTINEL, label: 'Custom' }),
    )

    const stored = JSON.parse(localStorage.getItem(BG_STORAGE_KEY)!)
    expect(stored.src).toBe(CUSTOM_SRC_SENTINEL)

    const customSrc = localStorage.getItem(CUSTOM_BG_KEY)
    expect(customSrc).toBe(dataUrl)
  })

  it('custom key is absent when no custom image has been uploaded', () => {
    expect(localStorage.getItem(CUSTOM_BG_KEY)).toBeNull()
  })

  it('removing the custom key makes the sentinel unresolvable', () => {
    localStorage.setItem(
      BG_STORAGE_KEY,
      JSON.stringify({ type: 'image', src: CUSTOM_SRC_SENTINEL, label: 'Custom' }),
    )
    localStorage.removeItem(CUSTOM_BG_KEY)

    const src = localStorage.getItem(CUSTOM_BG_KEY)
    expect(src).toBeNull()
  })

  it('overwriting custom key replaces the stored image', () => {
    localStorage.setItem(CUSTOM_BG_KEY, 'data:image/png;base64,first')
    localStorage.setItem(CUSTOM_BG_KEY, 'data:image/png;base64,second')
    expect(localStorage.getItem(CUSTOM_BG_KEY)).toBe('data:image/png;base64,second')
  })

  it('preset preference round-trips through JSON without sentinel', () => {
    const preset = { type: 'image', src: '/backgrounds/office.jpg', label: 'Office' }
    localStorage.setItem(BG_STORAGE_KEY, JSON.stringify(preset))
    const loaded = JSON.parse(localStorage.getItem(BG_STORAGE_KEY)!)
    expect(loaded).toEqual(preset)
    expect(loaded.src).not.toBe(CUSTOM_SRC_SENTINEL)
  })
})
