import Foundation
import WebKit

enum AllowedBridgeMethod: String, CaseIterable, Sendable {
    case getSessionToken
    case reportMetrics
    case closeView
}

struct ValidatedBridgeCall: Sendable, Equatable {
    let requestId: String
    let method: AllowedBridgeMethod
    let params: [String: AnyCodableValue]
}

enum BridgeRejection: Error, Sendable, Equatable {
    case nonMainFrame
    case untrustedOrigin(String)
    case invalidPayload
    case methodNotAllowed(String)

    var code: String {
        switch self {
        case .nonMainFrame: return "BRIDGE_NON_MAIN_FRAME"
        case .untrustedOrigin: return "BRIDGE_UNTRUSTED_ORIGIN"
        case .invalidPayload: return "BRIDGE_INVALID_PAYLOAD"
        case .methodNotAllowed: return "BRIDGE_METHOD_NOT_ALLOWED"
        }
    }

    var message: String {
        switch self {
        case .nonMainFrame:
            return "Panggilan bridge dari iframe tidak diizinkan."
        case let .untrustedOrigin(origin):
            return "Origin \(origin) tidak ada dalam allowlist."
        case .invalidPayload:
            return "Skema payload bridge tidak valid."
        case let .methodNotAllowed(method):
            return "Method '\(method)' ditolak oleh whitelist bridge native."
        }
    }
}

/// Memvalidasi dan mengeksekusi panggilan JS -> Native sesuai whitelist PRD (F7).
@MainActor
final class NativeBridgeHandler: NSObject, WKScriptMessageHandler {
    private let tokenManager: TokenManager
    private let metrics: MetricsCollector
    private let allowedHosts: Set<String>
    weak var webView: WKWebView?
    var onCloseRequested: (() -> Void)?

    init(
        tokenManager: TokenManager,
        metrics: MetricsCollector,
        allowedHosts: Set<String>
    ) {
        self.tokenManager = tokenManager
        self.metrics = metrics
        self.allowedHosts = Set(allowedHosts.map { $0.lowercased() })
        super.init()
    }

    static func validateMessage(
        body: Any,
        originHost: String?,
        isMainFrame: Bool,
        allowedHosts: Set<String>
    ) -> Result<ValidatedBridgeCall, BridgeRejection> {
        guard isMainFrame else {
            return .failure(.nonMainFrame)
        }

        let normalizedAllowed = Set(allowedHosts.map { $0.lowercased() })
        let host = originHost?.lowercased() ?? ""
        guard !host.isEmpty, normalizedAllowed.contains(host) else {
            return .failure(.untrustedOrigin(host.isEmpty ? "unknown" : host))
        }

        guard let dict = body as? [String: Any],
              let requestId = dict["requestId"] as? String, !requestId.isEmpty,
              let rawMethod = dict["method"] as? String, !rawMethod.isEmpty else {
            return .failure(.invalidPayload)
        }

        guard let method = AllowedBridgeMethod(rawValue: rawMethod) else {
            return .failure(.methodNotAllowed(rawMethod))
        }

        var safeParams: [String: AnyCodableValue] = [:]
        if let rawParams = dict["params"] as? [String: Any] {
            for (k, v) in rawParams {
                safeParams[k] = AnyCodableValue.fromRaw(v)
            }
        }

        return .success(ValidatedBridgeCall(requestId: requestId, method: method, params: safeParams))
    }

    func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
        let host = message.frameInfo.securityOrigin.host
        let isMainFrame = message.frameInfo.isMainFrame

        switch Self.validateMessage(
            body: message.body,
            originHost: host,
            isMainFrame: isMainFrame,
            allowedHosts: allowedHosts
        ) {
        case let .failure(rejection):
            metrics.recordSecurityEvent(
                source: "ApproachA.Bridge",
                decision: "REJECTED (\(rejection.code))",
                detail: rejection.message
            )
            if let dict = message.body as? [String: Any],
               let reqId = dict["requestId"] as? String {
                sendError(requestId: reqId, code: rejection.code, message: rejection.message)
            }
        case let .success(call):
            handleValidatedCall(call)
        }
    }

    private func handleValidatedCall(_ call: ValidatedBridgeCall) {
        switch call.method {
        case .getSessionToken:
            metrics.recordSecurityEvent(
                source: "ApproachA.Bridge",
                decision: "ALLOWED",
                detail: "Issued short-lived WebView token via getSessionToken"
            )
            Task {
                do {
                    let issued = try await tokenManager.mintShortLivedWebViewToken()
                    sendSuccess(requestId: call.requestId, data: [
                        "token": issued.token,
                        "expires_at": issued.expiresAt,
                        "ttl_seconds": issued.ttlSeconds,
                        "device_id": issued.deviceId
                    ])
                } catch {
                    sendError(requestId: call.requestId, code: "TOKEN_MINT_FAILED", message: error.localizedDescription)
                }
            }

        case .reportMetrics:
            if let tti = call.params["tti_ms"]?.doubleValue {
                metrics.recordTTI(approach: .webview, durationMs: tti)
            }
            if let rtt = call.params["bridge_rtt_ms"]?.doubleValue {
                metrics.recordBridgeRTT(durationMs: rtt)
            }
            if let fps = call.params["animation_fps"]?.doubleValue {
                let dropped = call.params["dropped_frames"]?.intValue ?? 0
                metrics.recordFPS(approach: .webview, fps: fps, droppedFrames: dropped)
            }
            sendSuccess(requestId: call.requestId, data: ["recorded": true])

        case .closeView:
            onCloseRequested?()
            sendSuccess(requestId: call.requestId, data: ["closed": true])
        }
    }

    private func sendSuccess(requestId: String, data: [String: Any]) {
        evaluateCallback(payload: [
            "requestId": requestId,
            "ok": true,
            "data": data
        ])
    }

    private func sendError(requestId: String, code: String, message: String) {
        evaluateCallback(payload: [
            "requestId": requestId,
            "ok": false,
            "error": [
                "code": code,
                "message": message
            ]
        ])
    }

    private func evaluateCallback(payload: [String: Any]) {
        guard let webView,
              let jsonData = try? JSONSerialization.data(withJSONObject: payload),
              let jsonString = String(data: jsonData, encoding: .utf8) else {
            return
        }
        let js = "window.__gachaBridgeResolve && window.__gachaBridgeResolve(\(jsonString));"
        webView.evaluateJavaScript(js, completionHandler: nil)
    }
}
