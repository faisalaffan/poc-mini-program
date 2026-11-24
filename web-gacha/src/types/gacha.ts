export type ClientApproach = 'webview' | 'native' | 'sdui'
export type Rarity = 'COMMON' | 'UNCOMMON' | 'RARE' | 'EPIC' | 'JACKPOT'
export type PinColor = 'blue' | 'orange' | 'red'

export interface RewardTier {
  id: string
  code: string
  name: string
  rarity: Rarity
  reward_points: number
  weight_bps: number
  probability_percent: number
  display_order: number
}

export interface BalanceResponse {
  user_id: string
  balance: number
  last_ledger_id: number
  updated_at: string
}

export interface ProbabilitiesResponse {
  table_version: string
  draw_cost: number
  total_weight_bps: number
  tiers: RewardTier[]
}

export interface DrawResponse {
  draw_id: string
  idempotency_key: string
  idempotency_replayed: boolean
  table_version: string
  draw_cost: number
  reward: RewardTier
  balance_before: number
  balance_after: number
  client_approach: ClientApproach
  created_at: string
}

export interface HistoryItem {
  draw_id: string
  table_version: string
  draw_cost: number
  reward_code: string
  reward_name: string
  rarity: Rarity
  reward_points: number
  balance_after: number
  client_approach: ClientApproach
  created_at: string
}

export interface VillageQuiz {
  id: string
  building_name: string
  title: string
  pin_color: PinColor
  x_ratio: number
  y_ratio: number
  reward_points: number
  question: string
  options: string[]
  completed: boolean
  display_order: number
}

export interface VillageStateResponse {
  user_id: string
  campaign_banner: string
  balance: number
  completed_count: number
  total_quizzes: number
  quizzes: VillageQuiz[]
  rules: string[]
}

export interface QuizSubmitResponse {
  submission_id: string
  quiz_id: string
  idempotency_key: string
  idempotency_replayed: boolean
  correct: boolean
  reward_points: number
  balance_before: number
  balance_after: number
  explanation: string
  client_approach: ClientApproach
  created_at: string
}

export interface SessionTokenPayload {
  token: string
  expires_at: string
  ttl_seconds: number
  device_id: string
}

export interface BridgeMetricsPayload {
  tti_ms?: number
  bridge_rtt_ms?: number
  animation_fps?: number
  dropped_frames?: number
  event?: string
}

export interface APIErrorEnvelope {
  error: {
    code: string
    message: string
    request_id?: string
  }
}
