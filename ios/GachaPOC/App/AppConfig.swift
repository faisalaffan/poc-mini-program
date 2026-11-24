import Foundation
import Observation

enum BackendEndpointPreset: String, CaseIterable, Identifiable, Sendable {
    case cloudflareHTTPS = "https://livin-village.faisalaffan.com"
    case k3sTailscaleHTTPS = "https://gacha-poc.singapura-parore.ts.net"
    case k3sNodePortHTTP = "http://vps-malang.singapura-parore.ts.net:30080"
    case macbookTailscaleHTTP = "http://macbook.singapura-parore.ts.net:8080"
    case localhost = "http://127.0.0.1:8080"

    var id: String { rawValue }

    var label: String {
        switch self {
        case .cloudflareHTTPS:
            return "Cloudflare (livin-village.faisalaffan.com)"
        case .k3sTailscaleHTTPS:
            return "k3s VPS (Tailscale HTTPS)"
        case .k3sNodePortHTTP:
            return "k3s VPS (NodePort :30080)"
        case .macbookTailscaleHTTP:
            return "MacBook Tailscale (:8080)"
        case .localhost:
            return "Localhost (Simulator)"
        }
    }
}

@MainActor
@Observable
final class AppConfig {
    var baseURLString: String = BackendEndpointPreset.cloudflareHTTPS.rawValue
    var allowedHosts: Set<String> = [
        "livin-village.faisalaffan.com",
        "gacha-poc.singapura-parore.ts.net",
        "vps-malang.singapura-parore.ts.net",
        "100.125.74.50",
        "macbook.singapura-parore.ts.net",
        "100.75.130.38",
        "127.0.0.1",
        "localhost"
    ]
    var pinnedSPKIHashes: Set<String> = []
    var webViewFallbackEnabled: Bool = false
    var sduiVariant: String = "default"
    var simulateUnsupportedSDUISchema: Bool = false
    var simulateTimeoutOnNextDraw: Bool = false

    var baseURL: URL {
        URL(string: baseURLString) ?? URL(string: BackendEndpointPreset.k3sTailscaleHTTPS.rawValue)!
    }

    var webViewEntryURL: URL {
        baseURL.appending(path: "webview/")
    }
}
