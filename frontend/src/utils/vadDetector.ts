/**
 * VadDetector — Voice Activity Detection using the Web Audio API.
 *
 * Computes the RMS amplitude of the microphone stream every 50 ms and emits:
 *   - `speaking` (boolean, debounced 400 ms): whether the user is speaking.
 *     Used to drive the remote-participant speaking border via WebSocket.
 *   - `level` (number 0–1, raw, every poll): instantaneous amplitude.
 *     Used to drive the local mic-button animation without any WS traffic.
 *
 * The 400 ms silence debounce prevents rapid on/off toggling of the speaking
 * border. The `level` value is NOT debounced so animations feel responsive.
 */

const SILENCE_DEBOUNCE_MS = 400
const SPEAKING_THRESHOLD = 0.015 // RMS amplitude 0–1
const SAMPLE_INTERVAL_MS = 50    // analyse every 50 ms → 20 Hz
const LEVEL_MAX_RMS = 0.08       // RMS at which level saturates to 1.0

export interface VadEvent {
  speaking: boolean // debounced
  level: number     // raw, 0–1
}

export type VadCallback = (event: VadEvent) => void

export class VadDetector {
  private audioCtx: AudioContext | null = null
  private analyser: AnalyserNode | null = null
  private source: MediaStreamAudioSourceNode | null = null
  private intervalId: ReturnType<typeof setInterval> | null = null
  private silenceTimer: ReturnType<typeof setTimeout> | null = null
  private speaking = false
  private readonly onUpdate: VadCallback

  constructor(onUpdate: VadCallback) {
    this.onUpdate = onUpdate
  }

  /** Attach to a MediaStream (microphone track). */
  start(stream: MediaStream): void {
    if (this.audioCtx) this.stop()

    this.audioCtx = new AudioContext()
    this.analyser = this.audioCtx.createAnalyser()
    this.analyser.fftSize = 512
    this.analyser.smoothingTimeConstant = 0.3

    this.source = this.audioCtx.createMediaStreamSource(stream)
    this.source.connect(this.analyser)

    const bufferLength = this.analyser.fftSize
    const dataArray = new Float32Array(bufferLength)

    this.intervalId = setInterval(() => {
      if (!this.analyser) return
      this.analyser.getFloatTimeDomainData(dataArray)

      // Compute RMS amplitude.
      let sum = 0
      for (let i = 0; i < bufferLength; i++) {
        sum += dataArray[i] * dataArray[i]
      }
      const rms = Math.sqrt(sum / bufferLength)

      // Normalised level — saturates at LEVEL_MAX_RMS.
      const level = Math.min(rms / LEVEL_MAX_RMS, 1)

      if (rms > SPEAKING_THRESHOLD) {
        if (this.silenceTimer !== null) {
          clearTimeout(this.silenceTimer)
          this.silenceTimer = null
        }
        const wasAlreadySpeaking = this.speaking
        if (!wasAlreadySpeaking) {
          this.speaking = true
        }
        this.onUpdate({ speaking: this.speaking, level })
      } else if (this.speaking && this.silenceTimer === null) {
        // Below threshold — start debounce, but still emit raw level.
        this.onUpdate({ speaking: true, level })
        this.silenceTimer = setTimeout(() => {
          this.speaking = false
          this.silenceTimer = null
          this.onUpdate({ speaking: false, level: 0 })
        }, SILENCE_DEBOUNCE_MS)
      } else if (!this.speaking) {
        // Emit level updates even when not speaking so the ring fades smoothly.
        this.onUpdate({ speaking: false, level })
      }
    }, SAMPLE_INTERVAL_MS)
  }

  /** Detach and clean up. */
  stop(): void {
    if (this.intervalId !== null) {
      clearInterval(this.intervalId)
      this.intervalId = null
    }
    if (this.silenceTimer !== null) {
      clearTimeout(this.silenceTimer)
      this.silenceTimer = null
    }
    this.source?.disconnect()
    this.audioCtx?.close()
    this.audioCtx = null
    this.analyser = null
    this.source = null
    this.speaking = false
  }

  isSpeaking(): boolean {
    return this.speaking
  }
}
