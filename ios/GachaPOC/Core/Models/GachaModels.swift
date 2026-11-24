import Foundation

enum ClientApproach: String, Codable, Sendable, CaseIterable, Identifiable {
    case webview
    case native
    case sdui

    var id: String { rawValue }

    var displayTitle: String {
        switch self {
        case .webview: return "A · WebView (Vue 3)"
        case .native: return "B · Native SwiftUI"
        case .sdui: return "C · Server-Driven UI"
        }
    }
}

enum Rarity: String, Codable, Sendable {
    case common = "COMMON"
    case uncommon = "UNCOMMON"
    case rare = "RARE"
    case epic = "EPIC"
    case jackpot = "JACKPOT"
}

struct SessionResponse: Codable, Sendable {
    let token: String
    let tokenType: String
    let expiresAt: String
    let ttlSeconds: Int
    let userId: String
    let deviceId: String

    enum CodingKeys: String, CodingKey {
        case token
        case tokenType = "token_type"
        case expiresAt = "expires_at"
        case ttlSeconds = "ttl_seconds"
        case userId = "user_id"
        case deviceId = "device_id"
    }
}

struct WebViewTokenResponse: Codable, Sendable {
    let token: String
    let expiresAt: String
    let ttlSeconds: Int
    let deviceId: String
    let scope: String

    enum CodingKeys: String, CodingKey {
        case token
        case expiresAt = "expires_at"
        case ttlSeconds = "ttl_seconds"
        case deviceId = "device_id"
        case scope
    }
}

struct BalanceResponse: Codable, Sendable, Equatable {
    let userId: String
    let balance: Int64
    let lastLedgerId: Int64
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case balance
        case lastLedgerId = "last_ledger_id"
        case updatedAt = "updated_at"
    }
}

struct RewardTier: Codable, Sendable, Equatable, Identifiable {
    let id: String
    let code: String
    let name: String
    let rarity: Rarity
    let rewardPoints: Int64
    let weightBps: Int
    let probabilityPercent: Double
    let displayOrder: Int

    enum CodingKeys: String, CodingKey {
        case id, code, name, rarity
        case rewardPoints = "reward_points"
        case weightBps = "weight_bps"
        case probabilityPercent = "probability_percent"
        case displayOrder = "display_order"
    }
}

struct ProbabilitiesResponse: Codable, Sendable, Equatable {
    let tableVersion: String
    let drawCost: Int64
    let totalWeightBps: Int
    let tiers: [RewardTier]

    enum CodingKeys: String, CodingKey {
        case tableVersion = "table_version"
        case drawCost = "draw_cost"
        case totalWeightBps = "total_weight_bps"
        case tiers
    }
}

struct DrawResponse: Codable, Sendable, Equatable {
    let drawId: String
    let idempotencyKey: String
    let idempotencyReplayed: Bool
    let tableVersion: String
    let drawCost: Int64
    let reward: RewardTier
    let balanceBefore: Int64
    let balanceAfter: Int64
    let clientApproach: ClientApproach
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case drawId = "draw_id"
        case idempotencyKey = "idempotency_key"
        case idempotencyReplayed = "idempotency_replayed"
        case tableVersion = "table_version"
        case drawCost = "draw_cost"
        case reward
        case balanceBefore = "balance_before"
        case balanceAfter = "balance_after"
        case clientApproach = "client_approach"
        case createdAt = "created_at"
    }
}

struct HistoryItem: Codable, Sendable, Equatable, Identifiable {
    let drawId: String
    let tableVersion: String
    let drawCost: Int64
    let rewardCode: String
    let rewardName: String
    let rarity: Rarity
    let rewardPoints: Int64
    let balanceAfter: Int64
    let clientApproach: ClientApproach
    let createdAt: String

    var id: String { drawId }

    enum CodingKeys: String, CodingKey {
        case drawId = "draw_id"
        case tableVersion = "table_version"
        case drawCost = "draw_cost"
        case rewardCode = "reward_code"
        case rewardName = "reward_name"
        case rarity
        case rewardPoints = "reward_points"
        case balanceAfter = "balance_after"
        case clientApproach = "client_approach"
        case createdAt = "created_at"
    }
}

struct HistoryResponse: Codable, Sendable {
    let items: [HistoryItem]
}

struct ReconcileReport: Codable, Sendable, Equatable {
    let userId: String
    let ledgerBalance: Int64
    let computedBalance: Int64
    let totalSeeded: Int64
    let totalQuizRewards: Int64
    let totalDrawCost: Int64
    let totalRewardPoints: Int64
    let totalDraws: Int
    let balanced: Bool

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case ledgerBalance = "ledger_balance"
        case computedBalance = "computed_balance"
        case totalSeeded = "total_seeded"
        case totalQuizRewards = "total_quiz_rewards"
        case totalDrawCost = "total_draw_cost"
        case totalRewardPoints = "total_reward_points"
        case totalDraws = "total_draws"
        case balanced
    }
}

struct APIErrorEnvelope: Codable, Sendable {
    struct Body: Codable, Sendable {
        let code: String
        let message: String
        let requestId: String?

        enum CodingKeys: String, CodingKey {
            case code, message
            case requestId = "request_id"
        }
    }
    let error: Body
}

enum PinColor: String, Codable, Sendable {
    case blue
    case orange
    case red
}

struct VillageQuiz: Codable, Sendable, Equatable, Identifiable {
    let id: String
    let buildingName: String
    let title: String
    let pinColor: PinColor
    let xRatio: Double
    let yRatio: Double
    let rewardPoints: Int64
    let question: String
    let options: [String]
    var completed: Bool
    let displayOrder: Int

    enum CodingKeys: String, CodingKey {
        case id, title, question, options, completed
        case buildingName = "building_name"
        case pinColor = "pin_color"
        case xRatio = "x_ratio"
        case yRatio = "y_ratio"
        case rewardPoints = "reward_points"
        case displayOrder = "display_order"
    }
}

struct VillageStateResponse: Codable, Sendable, Equatable {
    let userId: String
    let campaignBanner: String
    var balance: Int64
    var completedCount: Int
    let totalQuizzes: Int
    var quizzes: [VillageQuiz]
    let rules: [String]

    enum CodingKeys: String, CodingKey {
        case balance, quizzes, rules
        case userId = "user_id"
        case campaignBanner = "campaign_banner"
        case completedCount = "completed_count"
        case totalQuizzes = "total_quizzes"
    }
}

struct QuizSubmitResponse: Codable, Sendable, Equatable {
    let submissionId: String
    let quizId: String
    let idempotencyKey: String
    let idempotencyReplayed: Bool
    let correct: Bool
    let rewardPoints: Int64
    let balanceBefore: Int64
    let balanceAfter: Int64
    let explanation: String
    let clientApproach: ClientApproach
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case correct, explanation
        case submissionId = "submission_id"
        case quizId = "quiz_id"
        case idempotencyKey = "idempotency_key"
        case idempotencyReplayed = "idempotency_replayed"
        case rewardPoints = "reward_points"
        case balanceBefore = "balance_before"
        case balanceAfter = "balance_after"
        case clientApproach = "client_approach"
        case createdAt = "created_at"
    }
}
