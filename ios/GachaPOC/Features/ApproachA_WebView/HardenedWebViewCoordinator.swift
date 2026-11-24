import Foundation
import UIKit
import WebKit

@MainActor
final class HardenedWebViewCoordinator: NSObject, WKNavigationDelegate, UIScrollViewDelegate {
    private let pinner: CertificatePinner
    private let metrics: MetricsCollector
    private var navigationStart: CFAbsoluteTime?
    var onLoadError: ((String) -> Void)?
    var onLoadFinished: (() -> Void)?

    init(pinner: CertificatePinner, metrics: MetricsCollector) {
        self.pinner = pinner
        self.metrics = metrics
        super.init()
    }

    func markNavigationStart() {
        navigationStart = CFAbsoluteTimeGetCurrent()
    }

    func webView(
        _ webView: WKWebView,
        decidePolicyFor navigationAction: WKNavigationAction
    ) async -> WKNavigationActionPolicy {
        guard let url = navigationAction.request.url else {
            return .cancel
        }
        let scheme = url.scheme?.lowercased() ?? ""
        guard scheme == "https" || (pinner.allowLocalDebugPlaintext && scheme == "http") else {
            metrics.recordSecurityEvent(
                source: "ApproachA.WebView",
                decision: "BLOCKED_SCHEME",
                detail: "Blocked URL scheme: \(url.absoluteString)"
            )
            return .cancel
        }

        if pinner.isHostAllowed(url.host) {
            return .allow
        }

        metrics.recordSecurityEvent(
            source: "ApproachA.WebView",
            decision: "BLOCKED_ORIGIN",
            detail: "Blocked navigation outside allowlist: \(url.host ?? "unknown")"
        )
        return .cancel
    }

    func webView(
        _ webView: WKWebView,
        didReceive challenge: URLAuthenticationChallenge
    ) async -> (URLSession.AuthChallengeDisposition, URLCredential?) {
        pinner.evaluate(challenge: challenge)
    }

    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        if let start = navigationStart {
            let elapsedMs = (CFAbsoluteTimeGetCurrent() - start) * 1000.0
            metrics.recordTTI(approach: .webview, durationMs: (elapsedMs * 10).rounded() / 10)
            navigationStart = nil
        }
        onLoadFinished?()
    }

    func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
        onLoadError?(error.localizedDescription)
    }

    func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        onLoadError?(error.localizedDescription)
    }

    func viewForZooming(in scrollView: UIScrollView) -> UIView? {
        nil
    }
}
