import type { BackgroundOption } from '@/types/meeting'

/** The 9 emoji available in the reaction picker, in display order. */
export const EMOJI_LIST = ['❤️', '👍', '🎉', '👏', '😂', '😮', '😢', '🤔', '👎'] as const

/**
 * Inline SVG gradient as a data: URI — used for preset virtual backgrounds.
 * Inline so no /public/ asset has to ship; eliminates the 404 the previous
 * .jpg paths produced. The SVG renders a smooth two-stop linear gradient at
 * 1280×720 (the canvas pipeline's resolution).
 */
function gradient(stop1: string, stop2: string, angle = 135): string {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="1280" height="720" viewBox="0 0 1280 720"><defs><linearGradient id="g" gradientTransform="rotate(${angle})"><stop offset="0%" stop-color="${stop1}"/><stop offset="100%" stop-color="${stop2}"/></linearGradient></defs><rect width="1280" height="720" fill="url(#g)"/></svg>`
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`
}

/** Preset virtual backgrounds — inline SVG gradients (no network fetch). */
export const PRESET_IMAGES: Array<{ src: string; label: string }> = [
  { src: gradient('#1e293b', '#0f172a', 135), label: 'Slate' },
  { src: gradient('#312e81', '#7c3aed', 145), label: 'Violet' },
  { src: gradient('#0c4a6e', '#06b6d4', 135), label: 'Ocean' },
  { src: gradient('#064e3b', '#10b981', 145), label: 'Forest' },
  { src: gradient('#7c2d12', '#f59e0b', 145), label: 'Sunset' },
]

/** All background options in panel order. */
export const BACKGROUND_OPTIONS: BackgroundOption[] = [
  { type: 'none' },
  { type: 'blur', level: 'light' },
  { type: 'blur', level: 'heavy' },
  ...PRESET_IMAGES.map((img) => ({ type: 'image' as const, ...img })),
]
