// AudioWorklet processor: Float32 mono input → 16-kHz int16 LE PCM, emitted
// as 100 ms (1600-sample) chunks via port.postMessage.
//
// Served as a static asset under /pcm-worklet.js so the URL is identical in
// dev and production (the bundler doesn't try to transform it). Loaded by
// useASR via `await ctx.audioWorklet.addModule('/pcm-worklet.js')`. The
// AudioContext MUST be constructed with sampleRate: 16000; this worklet
// does not resample.

const FRAME_SAMPLES = 1600 // 100 ms @ 16 kHz

class PCMProcessor extends AudioWorkletProcessor {
  constructor() {
    super()
    this.carry = new Float32Array(0)
  }

  process(inputs) {
    const ch0 = inputs[0] && inputs[0][0]
    if (!ch0 || ch0.length === 0) return true

    const merged = new Float32Array(this.carry.length + ch0.length)
    merged.set(this.carry, 0)
    merged.set(ch0, this.carry.length)

    let cursor = 0
    while (merged.length - cursor >= FRAME_SAMPLES) {
      const view = merged.subarray(cursor, cursor + FRAME_SAMPLES)
      const out = new Int16Array(FRAME_SAMPLES)
      for (let i = 0; i < FRAME_SAMPLES; i++) {
        const s = Math.max(-1, Math.min(1, view[i]))
        out[i] = s < 0 ? Math.round(s * 0x8000) : Math.round(s * 0x7fff)
      }
      this.port.postMessage(out.buffer, [out.buffer])
      cursor += FRAME_SAMPLES
    }

    this.carry = merged.slice(cursor)
    return true
  }
}

registerProcessor('pcm-worklet', PCMProcessor)
