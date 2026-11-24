import { computed, ref } from 'vue'
import { GachaAPIClient, GachaAPIError } from '../api/client'
import { BridgeError, NativeBridgeClient, nativeBridge } from '../bridge/nativeBridge'
import type {
  DrawResponse,
  HistoryItem,
  ProbabilitiesResponse,
  QuizSubmitResponse,
  SessionTokenPayload,
  VillageQuiz,
  VillageStateResponse,
} from '../types/gacha'

function createUUID(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `idem-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export type ActiveModalType = 'quiz' | 'help' | 'rewards' | null

export function useVillage(bridge: NativeBridgeClient = nativeBridge, fetchImpl: typeof fetch = fetch.bind(globalThis)) {
  // Token hanya hidup di memori JS (closure), nol akses ke localStorage/sessionStorage/cookie.
  let memorySession: SessionTokenPayload | null = null
  let inFlightTokenPromise: Promise<SessionTokenPayload> | null = null

  const loading = ref(true)
  const submitting = ref(false)
  const village = ref<VillageStateResponse | null>(null)
  const probabilities = ref<ProbabilitiesResponse | null>(null)
  const history = ref<HistoryItem[]>([])

  const activeModal = ref<ActiveModalType>(null)
  const selectedQuiz = ref<VillageQuiz | null>(null)
  const selectedOptionIndex = ref<number | null>(null)
  const lastQuizResult = ref<QuizSubmitResponse | null>(null)
  const lastDrawResult = ref<DrawResponse | null>(null)

  const pendingQuizIdempotencyKey = ref<string | null>(null)
  const pendingDrawIdempotencyKey = ref<string | null>(null)
  const errorState = ref<{ code: string; message: string; retryable: boolean } | null>(null)
  const bridgeStatus = ref<{ rttMs: number | null; tokenTTL: number | null; probeNotice: string | null }>({
    rttMs: null,
    tokenTTL: null,
    probeNotice: null,
  })

  const quizzes = computed(() => village.value?.quizzes ?? [])
  const balance = computed(() => village.value?.balance ?? 0)
  const campaignBanner = computed(
    () =>
      village.value?.campaign_banner ??
      "Mainkan kuis Livin' Village & dapatkan hadiah menarik pada 1 Jun 2026 - 31 Dec 2026",
  )

  async function ensureSessionToken(forceRefresh = false): Promise<{ token: string; deviceId: string }> {
    const expiresSoon =
      memorySession !== null && Date.parse(memorySession.expires_at) - Date.now() < 15_000

    if (!forceRefresh && memorySession && !expiresSoon) {
      return { token: memorySession.token, deviceId: memorySession.device_id }
    }

    if (!inFlightTokenPromise) {
      inFlightTokenPromise = bridge
        .getSessionToken()
        .then(({ payload, rttMs }) => {
          memorySession = payload
          bridgeStatus.value = {
            ...bridgeStatus.value,
            rttMs,
            tokenTTL: payload.ttl_seconds,
          }
          void bridge.reportMetrics({ bridge_rtt_ms: rttMs, event: 'bridge_get_session_token' })
          return payload
        })
        .finally(() => {
          inFlightTokenPromise = null
        })
    }

    const session = await inFlightTokenPromise
    return { token: session.token, deviceId: session.device_id }
  }

  const api = new GachaAPIClient(ensureSessionToken, '', fetchImpl)

  async function loadAll(): Promise<void> {
    loading.value = true
    errorState.value = null
    const start = performance.now()
    try {
      const [state, probs, hist] = await Promise.all([
        api.getVillageState(),
        api.getProbabilities(),
        api.getHistory(10),
      ])
      village.value = state
      probabilities.value = probs
      history.value = hist.items ?? []

      if (selectedQuiz.value) {
        selectedQuiz.value = state.quizzes.find((q) => q.id === selectedQuiz.value?.id) ?? selectedQuiz.value
      }

      const ttiMs = Math.round((performance.now() - start) * 100) / 100
      void bridge.reportMetrics({ tti_ms: ttiMs, event: 'village_spa_interactive' })
    } catch (err) {
      applyError(err, true)
    } finally {
      loading.value = false
    }
  }

  function openQuiz(quiz: VillageQuiz): void {
    selectedQuiz.value = quiz
    selectedOptionIndex.value = null
    lastQuizResult.value = null
    pendingQuizIdempotencyKey.value = null
    errorState.value = null
    activeModal.value = 'quiz'
  }

  function openHelp(): void {
    errorState.value = null
    activeModal.value = 'help'
  }

  function openRewards(): void {
    errorState.value = null
    lastDrawResult.value = null
    activeModal.value = 'rewards'
  }

  function closeModal(): void {
    activeModal.value = null
  }

  async function submitSelectedQuiz(): Promise<void> {
    if (!selectedQuiz.value || selectedOptionIndex.value === null || submitting.value) return
    submitting.value = true
    errorState.value = null

    if (!pendingQuizIdempotencyKey.value) {
      pendingQuizIdempotencyKey.value = createUUID()
    }
    const key = pendingQuizIdempotencyKey.value

    try {
      const res = await api.submitQuiz(selectedQuiz.value.id, selectedOptionIndex.value, key)
      pendingQuizIdempotencyKey.value = null
      lastQuizResult.value = res
      if (village.value) {
        village.value = {
          ...village.value,
          balance: res.balance_after,
          quizzes: village.value.quizzes.map((q) =>
            q.id === res.quiz_id && res.correct ? { ...q, completed: true } : q,
          ),
        }
        selectedQuiz.value = village.value.quizzes.find((q) => q.id === res.quiz_id) ?? selectedQuiz.value
      }
      if (res.correct) {
        await loadAll()
      }
    } catch (err) {
      if (err instanceof GachaAPIError && err.status >= 400 && err.status < 500 && err.status !== 429) {
        pendingQuizIdempotencyKey.value = null
      }
      applyError(err, true)
    } finally {
      submitting.value = false
    }
  }

  async function drawGiftBox(): Promise<void> {
    if (submitting.value) return
    submitting.value = true
    errorState.value = null

    if (!pendingDrawIdempotencyKey.value) {
      pendingDrawIdempotencyKey.value = createUUID()
    }
    const key = pendingDrawIdempotencyKey.value

    try {
      const res = await api.draw(key, probabilities.value?.table_version)
      pendingDrawIdempotencyKey.value = null
      lastDrawResult.value = res
      if (village.value) {
        village.value = { ...village.value, balance: res.balance_after }
      }
      const hist = await api.getHistory(10)
      history.value = hist.items ?? []
    } catch (err) {
      if (err instanceof GachaAPIError) {
        if (err.code === 'INSUFFICIENT_BALANCE' || err.code === 'IDEMPOTENCY_PAYLOAD_MISMATCH') {
          pendingDrawIdempotencyKey.value = null
        }
      }
      applyError(err, true)
    } finally {
      submitting.value = false
    }
  }

  async function testUnauthorizedBridgeCall(): Promise<void> {
    try {
      await bridge.probeIllegalMethod('stealAllKeychainSecrets')
      bridgeStatus.value = {
        ...bridgeStatus.value,
        probeNotice: 'Peringatan: method ilegal tidak ditolak!',
      }
    } catch (err) {
      const msg = err instanceof BridgeError ? `${err.code}: ${err.message}` : String(err)
      bridgeStatus.value = {
        ...bridgeStatus.value,
        probeNotice: `Ditolak aman -> ${msg}`,
      }
    }
  }

  function applyError(err: unknown, retryable: boolean): void {
    if (err instanceof GachaAPIError) {
      errorState.value = { code: err.code, message: err.message, retryable }
    } else if (err instanceof BridgeError) {
      errorState.value = { code: err.code, message: err.message, retryable: true }
    } else {
      errorState.value = {
        code: 'NETWORK_ERROR',
        message: 'Koneksi gagal atau timeout. Ketuk Coba Lagi dengan Idempotency-Key yang sama.',
        retryable: true,
      }
    }
  }

  return {
    loading,
    submitting,
    village,
    quizzes,
    balance,
    campaignBanner,
    probabilities,
    history,
    activeModal,
    selectedQuiz,
    selectedOptionIndex,
    lastQuizResult,
    lastDrawResult,
    pendingQuizIdempotencyKey,
    pendingDrawIdempotencyKey,
    errorState,
    bridgeStatus,
    loadAll,
    openQuiz,
    openHelp,
    openRewards,
    closeModal,
    submitSelectedQuiz,
    drawGiftBox,
    testUnauthorizedBridgeCall,
  }
}
