import type { BackgroundOption } from '@/types/meeting'

/** The 9 emoji available in the reaction picker, in display order. */
export const EMOJI_LIST = ['❤️', '👍', '🎉', '👏', '😂', '😮', '😢', '🤔', '👎'] as const

/** Preset virtual backgrounds bundled under /public/backgrounds/. */
export const PRESET_IMAGES: Array<{ src: string; label: string }> = [
  { src: '/backgrounds/office.jpg',        label: 'Office' },
  { src: '/backgrounds/abstract-dark.jpg', label: 'Abstract' },
  { src: '/backgrounds/coffee-shop.jpg',   label: 'Café' },
  { src: '/backgrounds/nature.jpg',        label: 'Nature' },
  { src: '/backgrounds/space.jpg',         label: 'Space' },
]

/** All background options in panel order. */
export const BACKGROUND_OPTIONS: BackgroundOption[] = [
  { type: 'none' },
  { type: 'blur', level: 'light' },
  { type: 'blur', level: 'heavy' },
  ...PRESET_IMAGES.map((img) => ({ type: 'image' as const, ...img })),
]
