import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { VadDetector } from '@/utils/vadDetector'
import type { VadEvent } from '@/utils/vadDetector'

// ── AudioContext mock ──────────────────────────────────────────────────────────

class MockAnalyserNode {
  fftSize = 512
  smoothingTimeConstant = 0
  private _data: Float32Array | null = null

  setData(data: Float32Array) { this._data = data }

  getFloatTimeDomainData(buf: Float32Array) {
    if (this._data) buf.set(this._data.subarray(0, buf.length))
  }

  connect(_node: unknown) {}
}

class MockMediaStreamSourceNode {
  connect(_node: unknown) {}
  disconnect() {}
}

class MockAudioContext {
  analyser = new MockAnalyserNode()
  createAnalyser() { return this.analyser }
  createMediaStreamSource(_stream: MediaStream): MockMediaStreamSourceNode {
    return new MockMediaStreamSourceNode()
  }
  close() { return Promise.resolve() }
}

const fakeStream = { getTracks: () => [] } as unknown as MediaStream

// ─────────────────────────────────────────────────────────────────────────────

describe('VadDetector', () => {
  let ctx: MockAudioContext
  let onUpdate: ReturnType<typeof vi.fn>
  let detector: VadDetector

  beforeEach(() => {
    ctx = new MockAudioContext()
    vi.stubGlobal('AudioContext', vi.fn(() => ctx))
    onUpdate = vi.fn()
    detector = new VadDetector(onUpdate)
  })

  afterEach(() => {
    detector.stop()
    vi.restoreAllMocks()
  })

  it('starts not speaking', () => {
    expect(detector.isSpeaking()).toBe(false)
  })

  it('emits VadEvent with speaking:true and level>0 when amplitude exceeds threshold', async () => {
    ctx.analyser.setData(new Float32Array(512).fill(0.5))
    detector.start(fakeStream)

    await new Promise((r) => setTimeout(r, 100))

    expect(onUpdate).toHaveBeenCalled()
    const lastEvent: VadEvent = onUpdate.mock.calls.at(-1)![0]
    expect(lastEvent.speaking).toBe(true)
    expect(lastEvent.level).toBeGreaterThan(0)
    expect(lastEvent.level).toBeLessThanOrEqual(1)
    expect(detector.isSpeaking()).toBe(true)
  })

  it('emits VadEvent with speaking:false and level≈0 during silence', async () => {
    ctx.analyser.setData(new Float32Array(512).fill(0))
    detector.start(fakeStream)

    await new Promise((r) => setTimeout(r, 100))

    // May have emitted level updates — all should have speaking:false.
    for (const [event] of onUpdate.mock.calls) {
      expect((event as VadEvent).speaking).toBe(false)
    }
    expect(detector.isSpeaking()).toBe(false)
  })

  it('level saturates to 1.0 when amplitude is well above LEVEL_MAX_RMS', async () => {
    // fill(0.2) produces RMS ≈ 0.2, which is > LEVEL_MAX_RMS (0.08) → clamped to 1.0.
    ctx.analyser.setData(new Float32Array(512).fill(0.2))
    detector.start(fakeStream)

    await new Promise((r) => setTimeout(r, 100))

    const levels = onUpdate.mock.calls.map((args) => (args[0] as VadEvent).level)
    expect(levels.some((l) => l >= 1.0)).toBe(true)
  })

  it('emits level 0–1 range, never outside', async () => {
    // Extreme amplitude — should still clamp to 1.
    ctx.analyser.setData(new Float32Array(512).fill(1.0))
    detector.start(fakeStream)

    await new Promise((r) => setTimeout(r, 100))

    for (const [event] of onUpdate.mock.calls) {
      expect((event as VadEvent).level).toBeGreaterThanOrEqual(0)
      expect((event as VadEvent).level).toBeLessThanOrEqual(1)
    }
  })

  it('stop() prevents further callbacks', async () => {
    ctx.analyser.setData(new Float32Array(512).fill(0.5))
    detector.start(fakeStream)
    detector.stop()

    const countAfterStop = onUpdate.mock.calls.length
    await new Promise((r) => setTimeout(r, 150))

    expect(onUpdate.mock.calls.length).toBe(countAfterStop)
    expect(detector.isSpeaking()).toBe(false)
  })

  it('stop() resets isSpeaking to false', async () => {
    ctx.analyser.setData(new Float32Array(512).fill(0.5))
    detector.start(fakeStream)
    await new Promise((r) => setTimeout(r, 100))
    expect(detector.isSpeaking()).toBe(true)

    detector.stop()
    expect(detector.isSpeaking()).toBe(false)
  })

  it('emits speaking:true during the 400ms silence debounce window', async () => {
    // Phase 1: loud — detector starts speaking.
    ctx.analyser.setData(new Float32Array(512).fill(0.5))
    detector.start(fakeStream)
    await new Promise((r) => setTimeout(r, 100))
    expect(detector.isSpeaking()).toBe(true)

    // Phase 2: go silent — debounce starts; speaking should still be true immediately.
    ctx.analyser.setData(new Float32Array(512).fill(0))
    await new Promise((r) => setTimeout(r, 60))

    const lastEvent: VadEvent = onUpdate.mock.calls.at(-1)![0]
    // The last emitted event during the debounce window keeps speaking:true.
    expect(lastEvent.speaking).toBe(true)
  })

  it('emits speaking:false after the 400ms silence debounce expires', async () => {
    // Phase 1: loud.
    ctx.analyser.setData(new Float32Array(512).fill(0.5))
    detector.start(fakeStream)
    await new Promise((r) => setTimeout(r, 100))
    expect(detector.isSpeaking()).toBe(true)

    // Phase 2: silence for longer than SILENCE_DEBOUNCE_MS (400ms).
    ctx.analyser.setData(new Float32Array(512).fill(0))
    await new Promise((r) => setTimeout(r, 500))

    expect(detector.isSpeaking()).toBe(false)
    const lastEvent: VadEvent = onUpdate.mock.calls.at(-1)![0]
    expect(lastEvent.speaking).toBe(false)
    expect(lastEvent.level).toBe(0)
  })

  it('cancels the debounce timer when loud audio resumes during silence window', async () => {
    // Phase 1: loud.
    ctx.analyser.setData(new Float32Array(512).fill(0.5))
    detector.start(fakeStream)
    await new Promise((r) => setTimeout(r, 100))

    // Phase 2: brief silence — starts debounce.
    ctx.analyser.setData(new Float32Array(512).fill(0))
    await new Promise((r) => setTimeout(r, 60))

    // Phase 3: loud again — debounce should be cancelled, still speaking.
    ctx.analyser.setData(new Float32Array(512).fill(0.5))
    await new Promise((r) => setTimeout(r, 500))

    // Should still be speaking after 400ms (debounce was cancelled).
    expect(detector.isSpeaking()).toBe(true)
  })

  it('calling start() twice cleans up the previous session', async () => {
    ctx.analyser.setData(new Float32Array(512).fill(0.5))
    detector.start(fakeStream)
    await new Promise((r) => setTimeout(r, 60))
    const firstCount = onUpdate.mock.calls.length

    // Second start — should not double-fire callbacks.
    detector.start(fakeStream)
    await new Promise((r) => setTimeout(r, 60))
    const secondCount = onUpdate.mock.calls.length - firstCount

    // Each session runs one interval per 50ms, so secondCount should be ~1, not doubled.
    expect(secondCount).toBeLessThanOrEqual(3)
  })
})
