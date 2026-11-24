import { describe, expect, it, vi } from 'vitest'
import { NativeBridgeClient, type BridgeRequest } from '../bridge/nativeBridge'
import { useVillage } from '../composables/useVillage'

function setupMockBridge(): NativeBridgeClient {
  window.webkit = {
    messageHandlers: {
      gachaBridge: {
        postMessage: (msg: BridgeRequest) => {
          queueMicrotask(() => {
            if (msg.method === 'getSessionToken') {
              window.__gachaBridgeResolve?.({
                requestId: msg.requestId,
                ok: true,
                data: {
                  token: 'short-lived-jwt',
                  expires_at: new Date(Date.now() + 120_000).toISOString(),
                  ttl_seconds: 120,
                  device_id: 'ios-sim-test',
                },
              })
              return
            }
            window.__gachaBridgeResolve?.({ requestId: msg.requestId, ok: true, data: {} })
          })
        },
      },
    },
  }
  return new NativeBridgeClient(1000)
}

describe('useVillage composable', () => {
  it('preserves idempotency key on quiz submit timeout retry and avoids localStorage', async () => {
    localStorage.clear()
    const bridge = setupMockBridge()
    const capturedKeys: string[] = []
    let failFirstSubmit = true
    let serverBalance = 1000
    let serverCompleted = false

    const mockFetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (url.endsWith('/api/v1/village/state')) {
        return new Response(
          JSON.stringify({
            user_id: 'demo-user-001',
            campaign_banner: "Mainkan kuis Livin' Village",
            balance: serverBalance,
            completed_count: serverCompleted ? 1 : 0,
            total_quizzes: 1,
            rules: ['Aturan 1'],
            quizzes: [
              {
                id: 'quiz_tower',
                building_name: 'Menara Utama',
                title: 'Kuis Keamanan',
                pin_color: 'blue',
                x_ratio: 0.63,
                y_ratio: 0.44,
                reward_points: 150,
                question: 'Pertanyaan?',
                options: ['A', 'B'],
                completed: serverCompleted,
                display_order: 1,
              },
            ],
          }),
          { status: 200 },
        )
      }
      if (url.endsWith('/api/v1/gacha/probabilities')) {
        return new Response(JSON.stringify({ table_version: 'v1.0.0', draw_cost: 100, total_weight_bps: 10000, tiers: [] }), {
          status: 200,
        })
      }
      if (url.includes('/api/v1/gacha/history')) {
        return new Response(JSON.stringify({ items: [] }), { status: 200 })
      }
      if (url.includes('/api/v1/village/quizzes/quiz_tower/submit')) {
        const headers = init?.headers as Record<string, string>
        capturedKeys.push(headers['Idempotency-Key'])
        if (failFirstSubmit) {
          failFirstSubmit = false
          throw new Error('Timeout')
        }
        serverBalance = 1150
        serverCompleted = true
        return new Response(
          JSON.stringify({
            submission_id: 'sub-1',
            quiz_id: 'quiz_tower',
            idempotency_key: headers['Idempotency-Key'],
            idempotency_replayed: true,
            correct: true,
            reward_points: 150,
            balance_before: 1000,
            balance_after: 1150,
            explanation: 'Benar',
            client_approach: 'webview',
            created_at: new Date().toISOString(),
          }),
          { status: 200 },
        )
      }
      return new Response('{}', { status: 404 })
    }) as unknown as typeof fetch

    const v = useVillage(bridge, mockFetch)
    await v.loadAll()
    expect(v.quizzes.value).toHaveLength(1)

    v.openQuiz(v.quizzes.value[0])
    v.selectedOptionIndex.value = 1

    await v.submitSelectedQuiz()
    expect(v.pendingQuizIdempotencyKey.value).not.toBeNull()
    expect(v.errorState.value?.code).toBe('NETWORK_ERROR')

    await v.submitSelectedQuiz()
    expect(capturedKeys).toHaveLength(2)
    expect(capturedKeys[0]).toBe(capturedKeys[1])
    expect(v.pendingQuizIdempotencyKey.value).toBeNull()
    expect(v.balance.value).toBe(1150)
    expect(localStorage.length).toBe(0)
  })
})
