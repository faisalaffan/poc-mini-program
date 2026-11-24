import type { BridgeMetricsPayload, SessionTokenPayload } from '../types/gacha'

export interface BridgeRequest {
  requestId: string
  method: string
  params?: Record<string, unknown>
}

export interface BridgeResponse<T = unknown> {
  requestId: string
  ok: boolean
  data?: T
  error?: {
    code: string
    message: string
  }
}

interface PendingResolver {
  resolve: (value: unknown) => void
  reject: (reason: Error) => void
  timer: ReturnType<typeof setTimeout>
}

declare global {
  interface Window {
    webkit?: {
      messageHandlers?: {
        gachaBridge?: {
          postMessage: (message: BridgeRequest) => void
        }
      }
    }
    __gachaBridgeResolve?: (response: BridgeResponse) => void
  }
}

export class BridgeError extends Error {
  constructor(
    public readonly code: string,
    message: string,
  ) {
    super(message)
    this.name = 'BridgeError'
  }
}

export class NativeBridgeClient {
  private pending = new Map<string, PendingResolver>()
  private seq = 0
  private readonly timeoutMs: number

  constructor(timeoutMs = 5000) {
    this.timeoutMs = timeoutMs
    if (typeof window !== 'undefined') {
      window.__gachaBridgeResolve = (res) => this.handleResponse(res)
    }
  }

  isAvailable(): boolean {
    return typeof window !== 'undefined' && Boolean(window.webkit?.messageHandlers?.gachaBridge)
  }

  private handleResponse(res: BridgeResponse): void {
    const entry = this.pending.get(res.requestId)
    if (!entry) return
    clearTimeout(entry.timer)
    this.pending.delete(res.requestId)

    if (res.ok) {
      entry.resolve(res.data)
    } else {
      entry.reject(
        new BridgeError(res.error?.code ?? 'BRIDGE_ERROR', res.error?.message ?? 'Native bridge menolak permintaan.'),
      )
    }
  }

  invoke<T>(method: string, params?: Record<string, unknown>): Promise<T> {
    if (!this.isAvailable()) {
      return Promise.reject(
        new BridgeError('BRIDGE_UNAVAILABLE', 'Halaman ini harus dijalankan di dalam WebView Super-App iOS.'),
      )
    }

    const requestId = `br_${++this.seq}_${Date.now()}`
    return new Promise<T>((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(requestId)
        reject(new BridgeError('BRIDGE_TIMEOUT', `Timeout menunggu respons bridge untuk ${method}.`))
      }, this.timeoutMs)

      this.pending.set(requestId, {
        resolve: (value) => resolve(value as T),
        reject,
        timer,
      })

      window.webkit!.messageHandlers!.gachaBridge!.postMessage({
        requestId,
        method,
        params,
      })
    })
  }

  async getSessionToken(): Promise<{ payload: SessionTokenPayload; rttMs: number }> {
    const start = performance.now()
    const payload = await this.invoke<SessionTokenPayload>('getSessionToken')
    const rttMs = Math.round((performance.now() - start) * 100) / 100
    return { payload, rttMs }
  }

  async reportMetrics(metrics: BridgeMetricsPayload): Promise<void> {
    if (!this.isAvailable()) return
    try {
      await this.invoke('reportMetrics', metrics as Record<string, unknown>)
    } catch {
      // Telemetry tidak boleh menggagalkan alur user
    }
  }

  async probeIllegalMethod(methodName: string): Promise<void> {
    await this.invoke(methodName, { test: true })
  }
}

export const nativeBridge = new NativeBridgeClient()
