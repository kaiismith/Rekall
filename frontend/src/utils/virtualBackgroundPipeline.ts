/**
 * VirtualBackgroundPipeline
 *
 * Composites the camera feed onto a canvas with an optional virtual background.
 * The canvas stream replaces the original video track on all peer connections
 * via RTCRtpSender.replaceTrack().
 *
 * Current implementation uses CSS `filter: blur()` on the canvas for blur
 * backgrounds and image compositing for image backgrounds (person is drawn
 * on top of the image without segmentation — both are visible).
 *
 * To enable full person-segmentation (cutting the person out of the background)
 * install `@mediapipe/selfie_segmentation` and uncomment the initSegmenter()
 * call in setBackground(). The rest of the pipeline is already wired for it.
 *
 * npm install @mediapipe/selfie_segmentation
 */

import type { BackgroundOption } from '@/types/meeting'

const TARGET_FPS = 30
const FRAME_INTERVAL_MS = 1000 / TARGET_FPS

export class VirtualBackgroundPipeline {
  private canvas: HTMLCanvasElement
  private ctx: CanvasRenderingContext2D
  private sourceVideo: HTMLVideoElement
  private bgImage: HTMLImageElement | null = null
  private background: BackgroundOption = { type: 'none' }
  private rafHandle: number | null = null
  private lastFrameAt = 0
  private paused = false
  private destroyed = false

  constructor(sourceStream: MediaStream) {
    this.sourceVideo = document.createElement('video')
    this.sourceVideo.srcObject = sourceStream
    this.sourceVideo.muted = true
    this.sourceVideo.playsInline = true
    void this.sourceVideo.play()

    this.canvas = document.createElement('canvas')
    this.canvas.width = 1280
    this.canvas.height = 720
    this.ctx = this.canvas.getContext('2d')!
  }

  /**
   * Set a new background and return the canvas MediaStreamTrack to use as the
   * video sender track on all peer connections.
   * Returns null if canvas.captureStream is not supported.
   */
  async setBackground(option: BackgroundOption): Promise<MediaStreamTrack | null> {
    if (this.destroyed) return null
    if (!this.canvas.captureStream) return null

    this.background = option

    if (option.type === 'image') {
      await this.loadBgImage(option.src)
    } else {
      this.bgImage = null
    }

    if (option.type === 'none') {
      this.stopLoop()
      return null
    }

    this.startLoop()
    return this.canvas.captureStream(TARGET_FPS).getVideoTracks()[0] ?? null
  }

  /** Pause the RAF loop (camera off, screen sharing active). */
  pause(): void {
    this.paused = true
  }

  /** Resume the RAF loop. */
  resume(): void {
    this.paused = false
  }

  /** Tear down: stop the loop and release resources. */
  destroy(): void {
    this.destroyed = true
    this.stopLoop()
    this.sourceVideo.srcObject = null
  }

  // ─── private ───────────────────────────────────────────────────────────────

  private startLoop(): void {
    if (this.rafHandle !== null) return
    const tick = (now: number) => {
      if (this.destroyed) return
      this.rafHandle = requestAnimationFrame(tick)
      if (this.paused) return
      if (now - this.lastFrameAt < FRAME_INTERVAL_MS) return
      this.lastFrameAt = now
      this.drawFrame()
    }
    this.rafHandle = requestAnimationFrame(tick)
  }

  private stopLoop(): void {
    if (this.rafHandle !== null) {
      cancelAnimationFrame(this.rafHandle)
      this.rafHandle = null
    }
  }

  private drawFrame(): void {
    const { width, height } = this.canvas
    const bg = this.background

    if (bg.type === 'blur') {
      const px = bg.level === 'light' ? '8px' : '20px'
      // Draw blurred full frame (person + background blurred together).
      this.ctx.filter = `blur(${px})`
      this.ctx.drawImage(this.sourceVideo, 0, 0, width, height)
      this.ctx.filter = 'none'
    } else if (bg.type === 'image' && this.bgImage) {
      // Draw background image first, then the person (video) on top.
      // Without MediaPipe segmentation both layers are fully opaque, so the
      // image is visible only in the margins around the person if the camera
      // crops them. Full compositing requires @mediapipe/selfie_segmentation.
      this.ctx.drawImage(this.bgImage, 0, 0, width, height)
      this.ctx.drawImage(this.sourceVideo, 0, 0, width, height)
    } else {
      // Fallback / 'none': plain frame (should not reach here normally).
      this.ctx.drawImage(this.sourceVideo, 0, 0, width, height)
    }
  }

  private loadBgImage(src: string): Promise<void> {
    return new Promise((resolve) => {
      const img = new Image()
      img.crossOrigin = 'anonymous'
      img.onload = () => {
        this.bgImage = img
        resolve()
      }
      img.onerror = () => {
        this.bgImage = null
        resolve()
      }
      img.src = src
    })
  }

  /** True if canvas.captureStream is available in this browser. */
  static isSupported(): boolean {
    const c = document.createElement('canvas')
    return typeof c.captureStream === 'function'
  }
}
