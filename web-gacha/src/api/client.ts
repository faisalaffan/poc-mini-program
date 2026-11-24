import type {
  APIErrorEnvelope,
  BalanceResponse,
  DrawResponse,
  HistoryItem,
  ProbabilitiesResponse,
  QuizSubmitResponse,
  VillageStateResponse,
} from '../types/gacha'

export class GachaAPIError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
  ) {
    super(message)
    this.name = 'GachaAPIError'
  }
}

export type TokenProvider = (forceRefresh?: boolean) => Promise<{ token: string; deviceId: string }>

export class GachaAPIClient {
  constructor(
    private readonly getAuth: TokenProvider,
    private readonly baseUrl = '',
    private readonly fetchImpl: typeof fetch = fetch.bind(globalThis),
  ) {}

  private async request<T>(
    path: string,
    init: RequestInit = {},
    extraHeaders: Record<string, string> = {},
    allowRefresh = true,
  ): Promise<T> {
    const { token, deviceId } = await this.getAuth(false)
    const headers: Record<string, string> = {
      Authorization: `Bearer ${token}`,
      'X-Device-ID': deviceId,
      'X-Client-Approach': 'webview',
      ...extraHeaders,
    }
    if (init.body) {
      headers['Content-Type'] = 'application/json'
    }

    const res = await this.fetchImpl(`${this.baseUrl}${path}`, {
      ...init,
      headers,
      credentials: 'omit',
      cache: 'no-store',
    })

    if (res.status === 401 && allowRefresh) {
      await this.getAuth(true)
      return this.request<T>(path, init, extraHeaders, false)
    }

    if (!res.ok) {
      let code = 'HTTP_ERROR'
      let message = `Permintaan gagal (${res.status}).`
      try {
        const body = (await res.json()) as APIErrorEnvelope
        if (body?.error?.code) code = body.error.code
        if (body?.error?.message) message = body.error.message
      } catch {
        // fallback ke status text
      }
      throw new GachaAPIError(res.status, code, message)
    }

    return (await res.json()) as T
  }

  getBalance(): Promise<BalanceResponse> {
    return this.request<BalanceResponse>('/api/v1/gacha/balance')
  }

  getProbabilities(): Promise<ProbabilitiesResponse> {
    return this.request<ProbabilitiesResponse>('/api/v1/gacha/probabilities')
  }

  getHistory(limit = 15): Promise<{ items: HistoryItem[] }> {
    return this.request<{ items: HistoryItem[] }>(`/api/v1/gacha/history?limit=${limit}`)
  }

  draw(idempotencyKey: string, expectedTableVersion?: string): Promise<DrawResponse> {
    return this.request<DrawResponse>(
      '/api/v1/gacha/draw',
      {
        method: 'POST',
        body: JSON.stringify({ expected_table_version: expectedTableVersion ?? '' }),
      },
      {
        'Idempotency-Key': idempotencyKey,
      },
    )
  }

  getVillageState(): Promise<VillageStateResponse> {
    return this.request<VillageStateResponse>('/api/v1/village/state')
  }

  submitQuiz(quizId: string, selectedIndex: number, idempotencyKey: string): Promise<QuizSubmitResponse> {
    return this.request<QuizSubmitResponse>(
      `/api/v1/village/quizzes/${encodeURIComponent(quizId)}/submit`,
      {
        method: 'POST',
        body: JSON.stringify({ selected_index: selectedIndex }),
      },
      {
        'Idempotency-Key': idempotencyKey,
      },
    )
  }
}
