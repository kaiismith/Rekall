/**
 * AudioWorklet processor that converts the AudioContext's Float32 input
 * (range [-1, 1]) to 16-kHz int16 little-endian mono PCM and emits 100 ms
 * (1600-sample) chunks via `port.postMessage`.
 *
 * The host (useASR hook) wires `port.onmessage` to `ws.send(buffer)`. The
 * AudioContext MUST be created with `sampleRate: 16000`; this worklet does
 * not resample.
 *
 * Built as a standalone module (no React, no app imports) so the bundler
 * emits it as a separate chunk loadable by `audioWorklet.addModule(url)`.
 */

// AudioWorkletGlobalScope augmentations — vitest's typescript checker won't
// see these without `lib.dom` enabled, so we declare the bare minimum.
/* eslint-disable @typescript-eslint/no-explicit-any */
declare const sampleRate: number
declare class AudioWorkletProcessor {
  port: MessagePort
  constructor()
  process(inputs: Float32Array[][], outputs: Float32Array[][], parameters: Record<string, Float32Array>): boolean
}
declare function registerProcessor(name: string, processorCtor: any): void
/* eslint-enable @typescript-eslint/no-explicit-any */

const FRAME_SAMPLES = 1600 // 100 ms @ 16 kHz

class PCMProcessor extends AudioWorkletProcessor {
  private carry: Float32Array = new Float32Array(0)

  process(inputs: Float32Array[][]): boolean {
    const ch0 = inputs[0]?.[0]
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
      // Transfer the underlying buffer for zero-copy hand-off.
      this.port.postMessage(out.buffer, [out.buffer])
      cursor += FRAME_SAMPLES
    }

    this.carry = merged.slice(cursor)
    return true
  }
}

registerProcessor('pcm-worklet', PCMProcessor)
