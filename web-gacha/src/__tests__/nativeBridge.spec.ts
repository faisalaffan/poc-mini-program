import { describe, expect, it } from 'vitest'
import { BridgeError, NativeBridgeClient, type BridgeRequest } from '../bridge/nativeBridge'

describe('NativeBridgeClient', () => {
  it('resolves getSessionToken and measures bridge RTT', async () => {
    const posted: BridgeRequest[] = []
    window.webkit = {
      messageHandlers: {
        gachaBridge: {
          postMessage: (msg: BridgeRequest) => {
            posted.push(msg)
            queueMicrotask(() => {
              window.__gachaBridgeResolve?.({
                requestId: msg.requestId,
                ok: true,
                data: {
                  token: 'wv-short-token',
                  expires_at: new Date(Date.now() + 120_000).toISOString(),
                  ttl_seconds: 120,
                  device_id: 'ios-sim-1',
                },
              })
            })
          },
        },
      },
    }

    const client = new NativeBridgeClient(1000)
    const result = await client.getSessionToken()

    expect(posted).toHaveLength(1)
    expect(posted[0].method).toBe('getSessionToken')
    expect(result.payload.token).toBe('wv-short-token')
    expect(result.rttMs).toBeGreaterThanOrEqual(0)
  })

  it('rejects with BridgeError when native rejects non-whitelisted method', async () => {
    window.webkit = {
      messageHandlers: {
        gachaBridge: {
          postMessage: (msg: BridgeRequest) => {
            queueMicrotask(() => {
              window.__gachaBridgeResolve?.({
                requestId: msg.requestId,
                ok: false,
                error: {
                  code: 'BRIDGE_METHOD_NOT_ALLOWED',
                  message: 'Method not in whitelist',
                },
              })
            })
          },
        },
      },
    }

    const client = new NativeBridgeClient(1000)
    await expect(client.probeIllegalMethod('stealAllKeychainSecrets')).rejects.toBeInstanceOf(BridgeError)
  })
})
