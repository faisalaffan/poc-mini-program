import Foundation

enum GachaAPIError: LocalizedError, Sendable, Equatable {
    case missingSession
    case invalidURL
    case serverError(status: Int, code: String, message: String)
    case networkFailure(String)
    case decodingFailure(String)

    var errorDescription: String? {
        switch self {
        case .missingSession:
            return "Sesi POC belum siap. Ketuk Muat Ulang untuk membuat sesi."
        case .invalidURL:
            return "URL backend tidak valid."
        case let .serverError(_, _, message):
            return message
        case let .networkFailure(message):
            return message
        case let .decodingFailure(message):
            return "Gagal membaca respons server: \(message)"
        }
    }

    var errorCode: String {
        switch self {
        case .missingSession: return "MISSING_SESSION"
        case .invalidURL: return "INVALID_URL"
        case let .serverError(_, code, _): return code
        case .networkFailure: return "NETWORK_ERROR"
        case .decodingFailure: return "DECODING_ERROR"
        }
    }

    var isRetryable: Bool {
        switch self {
        case .networkFailure, .missingSession:
            return true
        case let .serverError(status, _, _):
            return status >= 500 || status == 429
        case .invalidURL, .decodingFailure:
            return false
        }
    }
}
