import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { VirtualBackgroundPipeline } from '@/utils/virtualBackgroundPipeline'

// ── Canvas / captureStream mock ───────────────────────────────────────────────

function makeFakeTrack(): MediaStreamTrack {
  return { kind: 'video', stop: vi.fn() } as unknown as MediaStreamTrack
}

function makeFakeCaptureStream(track: MediaStreamTrack) {
  return { getVideoTracks: () => [track] } as unknown as MediaStream
}

function setupCanvasMock(supported = true) {
  const fakeTrack = makeFakeTrack()
  const fakeCtx = {
    filter: 'none',
    drawImage: vi.fn(),
  }

  const fakeCanvas = {
    width: 0,
    height: 0,
    getContext: vi.fn(() => fakeCtx),
    captureStream: supported
      ? vi.fn(() => makeFakeCaptureStream(fakeTrack))
      : undefined,
  }

  vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
    if (tag === 'canvas') return fakeCanvas as unknown as HTMLCanvasElement
    // video element
    const video = {
      srcObject: null,
      muted: false,
      playsInline: false,
      play: vi.fn(() => Promise.resolve()),
    }
    return video as unknown as HTMLVideoElement
  })

  return { fakeCanvas, fakeCtx, fakeTrack }
}

// ── Fake MediaStream ──────────────────────────────────────────────────────────

function fakeStream(): MediaStream {
  return { getTracks: () => [] } as unknown as MediaStream
}

// ─────────────────────────────────────────────────────────────────────────────

describe('VirtualBackgroundPipeline.isSupported()', () => {
  afterEach(() => vi.restoreAllMocks())

  it('returns true when canvas.captureStream exists', () => {
    setupCanvasMock(true)
    expect(VirtualBackgroundPipeline.isSupported()).toBe(true)
  })

  it('returns false when canvas.captureStream is absent', () => {
    setupCanvasMock(false)
    expect(VirtualBackgroundPipeline.isSupported()).toBe(false)
  })
})

describe('VirtualBackgroundPipeline instance', () => {
  let mocks: ReturnType<typeof setupCanvasMock>

  beforeEach(() => {
    vi.useFakeTimers()
    mocks = setupCanvasMock(true)
    vi.stubGlobal('requestAnimationFrame', vi.fn())
    vi.stubGlobal('cancelAnimationFrame', vi.fn())
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('setBackground("none") returns null and does not start a RAF loop', async () => {
    const pipeline = new VirtualBackgroundPipeline(fakeStream())
    const track = await pipeline.setBackground({ type: 'none' })
    expect(track).toBeNull()
    expect(requestAnimationFrame).not.toHaveBeenCalled()
  })

  it('setBackground("blur") returns a MediaStreamTrack', async () => {
    const pipeline = new VirtualBackgroundPipeline(fakeStream())
    const track = await pipeline.setBackground({ type: 'blur', level: 'light' })
    expect(track).not.toBeNull()
    expect(track?.kind).toBe('video')
  })

  it('setBackground("blur" heavy) also returns a track', async () => {
    const pipeline = new VirtualBackgroundPipeline(fakeStream())
    const track = await pipeline.setBackground({ type: 'blur', level: 'heavy' })
    expect(track).not.toBeNull()
  })

  it('setBackground("image") returns a track when image loads', async () => {
    // Stub Image so onload fires immediately.
    const OrigImage = globalThis.Image
    class FakeImage {
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      crossOrigin = ''
      set src(_v: string) { setTimeout(() => this.onload?.(), 0) }
    }
    vi.stubGlobal('Image', FakeImage)

    const pipeline = new VirtualBackgroundPipeline(fakeStream())
    const trackPromise = pipeline.setBackground({ type: 'image', src: '/test.jpg', label: 'Test' })
    await vi.runAllTimersAsync()
    const track = await trackPromise

    expect(track).not.toBeNull()
    vi.stubGlobal('Image', OrigImage)
  })

  it('setBackground("image") returns a track even when image fails to load', async () => {
    class FakeImageError {
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      crossOrigin = ''
      set src(_v: string) { setTimeout(() => this.onerror?.(), 0) }
    }
    vi.stubGlobal('Image', FakeImageError)

    const pipeline = new VirtualBackgroundPipeline(fakeStream())
    const trackPromise = pipeline.setBackground({ type: 'image', src: '/bad.jpg', label: 'Bad' })
    await vi.runAllTimersAsync()
    const track = await trackPromise

    // Pipeline still starts; track is returned even without bgImage.
    expect(track).not.toBeNull()
  })

  it('setBackground after destroy() returns null', async () => {
    const pipeline = new VirtualBackgroundPipeline(fakeStream())
    pipeline.destroy()
    const track = await pipeline.setBackground({ type: 'blur', level: 'light' })
    expect(track).toBeNull()
  })

  it('destroy() calls cancelAnimationFrame', async () => {
    vi.mocked(requestAnimationFrame).mockReturnValue(42)
    const pipeline = new VirtualBackgroundPipeline(fakeStream())
    await pipeline.setBackground({ type: 'blur', level: 'light' })
    pipeline.destroy()
    expect(cancelAnimationFrame).toHaveBeenCalledWith(42)
  })

  describe('drawFrame — blur', () => {
    function makeRafThatFiresOnce() {
      // First call registers the loop; second call fires the tick with a timestamp
      // large enough to pass the FRAME_INTERVAL_MS (33ms) guard.
      let calls = 0
      const ref: { cb: FrameRequestCallback | null } = { cb: null }
      vi.mocked(requestAnimationFrame).mockImplementation((cb) => {
        calls++
        ref.cb = cb
        if (calls === 2) ref.cb(1000)
        return calls
      })
      return { fire: () => ref.cb?.(1000) }
    }

    it('sets blur filter and calls drawImage once for light blur', async () => {
      const raf = makeRafThatFiresOnce()
      const pipeline = new VirtualBackgroundPipeline(fakeStream())
      await pipeline.setBackground({ type: 'blur', level: 'light' })
      raf.fire()

      expect(mocks.fakeCtx.filter).toBe('none') // reset after draw
      expect(mocks.fakeCtx.drawImage).toHaveBeenCalledTimes(1)
    })

    it('sets blur filter and calls drawImage once for heavy blur', async () => {
      const raf = makeRafThatFiresOnce()
      const pipeline = new VirtualBackgroundPipeline(fakeStream())
      await pipeline.setBackground({ type: 'blur', level: 'heavy' })
      raf.fire()

      expect(mocks.fakeCtx.drawImage).toHaveBeenCalledTimes(1)
    })
  })

  describe('drawFrame — image', () => {
    beforeEach(() => {
      class FakeImage {
        onload: (() => void) | null = null
        onerror: (() => void) | null = null
        crossOrigin = ''
        set src(_v: string) { setTimeout(() => this.onload?.(), 0) }
      }
      vi.stubGlobal('Image', FakeImage)
    })

    it('calls drawImage twice when bgImage is loaded (bg + video)', async () => {
      const ref: { cb: FrameRequestCallback | null } = { cb: null }
      vi.mocked(requestAnimationFrame).mockImplementation((cb) => {
        ref.cb = cb
        return 1
      })

      const pipeline = new VirtualBackgroundPipeline(fakeStream())
      const trackPromise = pipeline.setBackground({ type: 'image', src: '/bg.jpg', label: 'BG' })
      await vi.runAllTimersAsync()
      await trackPromise

      // Fire one frame tick past the interval guard.
      ref.cb?.(1000)

      expect(mocks.fakeCtx.drawImage).toHaveBeenCalledTimes(2)
    })
  })

  // ── pause / resume ───────────────────────────────────────────────────────────

  describe('pause() and resume()', () => {
    /** RAF mock that stores the latest callback and lets the test fire it manually. */
    function makeControllableRaf() {
      const ref: { cb: FrameRequestCallback | null } = { cb: null }
      vi.mocked(requestAnimationFrame).mockImplementation((cb) => {
        ref.cb = cb
        return 1
      })
      return {
        /** Fire a tick with timestamp `now` (default far past the 33ms interval guard). */
        fire: (now = 1000) => ref.cb?.(now),
      }
    }

    it('pause() prevents drawImage from being called on the next tick', async () => {
      const raf = makeControllableRaf()
      const pipeline = new VirtualBackgroundPipeline(fakeStream())
      await pipeline.setBackground({ type: 'blur', level: 'light' })

      pipeline.pause()
      raf.fire()

      expect(mocks.fakeCtx.drawImage).not.toHaveBeenCalled()
    })

    it('resume() re-enables drawing after pause', async () => {
      const raf = makeControllableRaf()
      const pipeline = new VirtualBackgroundPipeline(fakeStream())
      await pipeline.setBackground({ type: 'blur', level: 'light' })

      pipeline.pause()
      raf.fire(1000)
      expect(mocks.fakeCtx.drawImage).not.toHaveBeenCalled()

      pipeline.resume()
      // Fire a second tick with a new timestamp so the interval guard passes.
      raf.fire(2000)
      expect(mocks.fakeCtx.drawImage).toHaveBeenCalledTimes(1)
    })

    it('does not start a second RAF loop when setBackground is called twice', async () => {
      const pipeline = new VirtualBackgroundPipeline(fakeStream())
      await pipeline.setBackground({ type: 'blur', level: 'light' })
      const callsAfterFirst = vi.mocked(requestAnimationFrame).mock.calls.length

      // Second call — rafHandle is already set, startLoop should no-op.
      await pipeline.setBackground({ type: 'blur', level: 'heavy' })
      expect(vi.mocked(requestAnimationFrame).mock.calls.length).toBe(callsAfterFirst)
    })
  })

  // ── drawFrame fallback branch ─────────────────────────────────────────────────

  describe('drawFrame — fallback (image bg with failed image load)', () => {
    it('calls drawImage once with the video source when bgImage is null', async () => {
      class FakeImageError {
        onload: (() => void) | null = null
        onerror: (() => void) | null = null
        crossOrigin = ''
        set src(_v: string) { setTimeout(() => this.onerror?.(), 0) }
      }
      vi.stubGlobal('Image', FakeImageError)

      const ref: { cb: FrameRequestCallback | null } = { cb: null }
      vi.mocked(requestAnimationFrame).mockImplementation((cb) => {
        ref.cb = cb
        return 1
      })

      const pipeline = new VirtualBackgroundPipeline(fakeStream())
      const trackPromise = pipeline.setBackground({ type: 'image', src: '/bad.jpg', label: 'Bad' })
      await vi.runAllTimersAsync()
      await trackPromise

      // bgImage is null (load failed) → else-fallback draws sourceVideo once.
      ref.cb?.(1000)
      expect(mocks.fakeCtx.drawImage).toHaveBeenCalledTimes(1)
    })
  })

  it('returns null when captureStream returns no video tracks (?? null branch)', async () => {
    // Override captureStream to return an empty track list.
    vi.restoreAllMocks()
    const fakeCtx = { filter: 'none', drawImage: vi.fn() }
    const fakeCanvas = {
      width: 0, height: 0,
      getContext: vi.fn(() => fakeCtx),
      captureStream: vi.fn(() => ({ getVideoTracks: () => [] })),
    }
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      if (tag === 'canvas') return fakeCanvas as unknown as HTMLCanvasElement
      return { srcObject: null, muted: false, playsInline: false, play: vi.fn(() => Promise.resolve()) } as unknown as HTMLVideoElement
    })
    vi.stubGlobal('requestAnimationFrame', vi.fn())
    vi.stubGlobal('cancelAnimationFrame', vi.fn())

    const pipeline = new VirtualBackgroundPipeline(fakeStream())
    const track = await pipeline.setBackground({ type: 'blur', level: 'light' })
    expect(track).toBeNull()
  })

  it('returns null when captureStream is not supported', async () => {
    vi.restoreAllMocks()
    setupCanvasMock(false)
    vi.stubGlobal('requestAnimationFrame', vi.fn())
    vi.stubGlobal('cancelAnimationFrame', vi.fn())
    const pipeline = new VirtualBackgroundPipeline(fakeStream())
    const track = await pipeline.setBackground({ type: 'blur', level: 'light' })
    expect(track).toBeNull()
  })

  it('tick exits early without drawing when pipeline is destroyed mid-flight', async () => {
    const ref: { cb: FrameRequestCallback | null } = { cb: null }
    vi.mocked(requestAnimationFrame).mockImplementation((cb) => {
      ref.cb = cb
      return 1
    })

    const pipeline = new VirtualBackgroundPipeline(fakeStream())
    await pipeline.setBackground({ type: 'blur', level: 'light' })

    // Destroy before the stored tick fires — exercises the `if (destroyed) return` guard.
    pipeline.destroy()
    ref.cb?.(1000)

    expect(mocks.fakeCtx.drawImage).not.toHaveBeenCalled()
  })
})
