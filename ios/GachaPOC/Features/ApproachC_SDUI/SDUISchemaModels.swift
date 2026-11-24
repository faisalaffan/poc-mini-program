import Foundation

enum AnyCodableValue: Codable, Sendable, Equatable {
    case string(String)
    case int(Int)
    case double(Double)
    case bool(Bool)
    case array([AnyCodableValue])
    case object([String: AnyCodableValue])
    case null

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() {
            self = .null
        } else if let b = try? container.decode(Bool.self) {
            self = .bool(b)
        } else if let i = try? container.decode(Int.self) {
            self = .int(i)
        } else if let d = try? container.decode(Double.self) {
            self = .double(d)
        } else if let s = try? container.decode(String.self) {
            self = .string(s)
        } else if let arr = try? container.decode([AnyCodableValue].self) {
            self = .array(arr)
        } else if let obj = try? container.decode([String: AnyCodableValue].self) {
            self = .object(obj)
        } else {
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Unsupported JSON value")
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .null: try container.encodeNil()
        case let .bool(b): try container.encode(b)
        case let .int(i): try container.encode(i)
        case let .double(d): try container.encode(d)
        case let .string(s): try container.encode(s)
        case let .array(a): try container.encode(a)
        case let .object(o): try container.encode(o)
        }
    }

    var stringValue: String? {
        if case let .string(s) = self { return s }
        return nil
    }

    var boolValue: Bool? {
        if case let .bool(b) = self { return b }
        return nil
    }

    var intValue: Int? {
        switch self {
        case let .int(i): return i
        case let .double(d): return Int(d)
        default: return nil
        }
    }

    var doubleValue: Double? {
        switch self {
        case let .double(d): return d
        case let .int(i): return Double(i)
        default: return nil
        }
    }

    var arrayValue: [AnyCodableValue]? {
        if case let .array(arr) = self { return arr }
        return nil
    }

    static func fromRaw(_ raw: Any) -> AnyCodableValue {
        switch raw {
        case let b as Bool: return .bool(b)
        case let i as Int: return .int(i)
        case let d as Double: return .double(d)
        case let s as String: return .string(s)
        case let arr as [Any]: return .array(arr.map(fromRaw))
        case let dict as [String: Any]:
            var mapped: [String: AnyCodableValue] = [:]
            for (k, v) in dict {
                mapped[k] = fromRaw(v)
            }
            return .object(mapped)
        default:
            return .null
        }
    }
}

enum SDUIComponentType: String, CaseIterable, Sendable, Equatable {
    case text
    case button
    case balanceCard = "balance_card"
    case probabilityList = "probability_list"
    case gachaBox = "gacha_box"
    case historyList = "history_list"

    // Komponen Village Quiz Full-Screen
    case floatingBanner = "floating_banner"
    case floatingAction = "floating_action"
    case mapPin = "map_pin"
}

enum SDUIActionType: String, CaseIterable, Sendable, Equatable {
    case draw
    case refresh
    case toggleHistory = "toggle_history"

    // Aksi whitelist Village Quiz
    case openQuiz = "open_quiz"
    case openHelp = "open_help"
    case openRewards = "open_rewards"
}

struct SDUINodePayload: Codable, Sendable, Equatable, Identifiable {
    let id: String
    let type: String
    let props: [String: AnyCodableValue]?
    let action: String?
}

struct SDUIScreenPayload: Codable, Sendable, Equatable {
    let schemaVersion: Int
    let screenId: String
    let variant: String
    let navigationTitle: String?
    let backgroundAsset: String?
    let tableVersion: String
    let supportedActions: [String]
    let components: [SDUINodePayload]
    let generatedAt: String

    enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case screenId = "screen_id"
        case variant
        case navigationTitle = "navigation_title"
        case backgroundAsset = "background_asset"
        case tableVersion = "table_version"
        case supportedActions = "supported_actions"
        case components
        case generatedAt = "generated_at"
    }
}

struct ValidatedSDUINode: Sendable, Equatable, Identifiable {
    let id: String
    let type: SDUIComponentType
    let props: [String: AnyCodableValue]
    let action: SDUIActionType?
}

struct ValidatedSDUIScreen: Sendable, Equatable {
    let schemaVersion: Int
    let screenId: String
    let variant: String
    let navigationTitle: String?
    let backgroundAsset: String?
    let tableVersion: String
    let nodes: [ValidatedSDUINode]
}
