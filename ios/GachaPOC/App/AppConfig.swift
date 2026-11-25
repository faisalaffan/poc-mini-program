import Foundation
import Observation

enum BackendEndpointPreset: String, CaseIterable, Identifiable, Sendable {
    case cloudflareHTTPS = "https://livin-village.faisalaffan.com"
    case macbookTailscaleHTTP = "http://macbook.singapura-parore.ts.net:8080"

    var id: String { rawValue }

    var label: String {
        switch self {
        case .cloudflareHTTPS:
            return "Production (livin-village.faisalaffan.com)"
        case .macbookTailscaleHTTP:
            return "MacBook Tailscale (:8080)"
        }
    }
}

@MainActor
@Observable
final class AppConfig {
    private static let baseURLDefaultsKey = "gacha.selected_base_url"

    var baseURLString: String {
        didSet {
            UserDefaults.standard.set(baseURLString, forKey: Self.baseURLDefaultsKey)
        }
    }
    var allowedHosts: Set<String> = [
        "livin-village.faisalaffan.com",
        "macbook.singapura-parore.ts.net",
        "100.75.130.38"
    ]
    var pinnedSPKIHashes: Set<String> = []
    var webViewFallbackEnabled: Bool = false
    var sduiVariant: String = "default"
    var simulateUnsupportedSDUISchema: Bool = false
    var simulateTimeoutOnNextDraw: Bool = false

    init() {
        if let saved = UserDefaults.standard.string(forKey: Self.baseURLDefaultsKey),
           BackendEndpointPreset(rawValue: saved) != nil {
            self.baseURLString = saved
        } else {
            self.baseURLString = BackendEndpointPreset.cloudflareHTTPS.rawValue
        }
    }

    var baseURL: URL {
        URL(string: baseURLString) ?? URL(string: BackendEndpointPreset.cloudflareHTTPS.rawValue)!
    }

    var webViewEntryURL: URL {
        baseURL.appending(path: "webview/")
    }
}
