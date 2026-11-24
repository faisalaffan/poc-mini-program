import SwiftUI
import WebKit

/// Subclass WKWebView untuk memblokir penuh menu tooltip Copy/Paste/LookUp/Share bawaan iOS.
final class NoCalloutWebView: WKWebView {
    override func canPerformAction(_ action: Selector, withSender sender: Any?) -> Bool {
        false
    }
}

struct GachaWebViewContainer: UIViewRepresentable {
    let url: URL
    let reloadToken: UUID
    let tokenManager: TokenManager
    let pinner: CertificatePinner
    let metrics: MetricsCollector
    let onError: (String?) -> Void
    let onCloseRequested: () -> Void

    func makeCoordinator() -> ContainerCoordinator {
        ContainerCoordinator(
            bridge: NativeBridgeHandler(
                tokenManager: tokenManager,
                metrics: metrics,
                allowedHosts: pinner.allowedHosts
            ),
            navDelegate: HardenedWebViewCoordinator(pinner: pinner, metrics: metrics)
        )
    }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.websiteDataStore = .nonPersistent()
        config.preferences.javaScriptCanOpenWindowsAutomatically = false
        config.defaultWebpagePreferences.allowsContentJavaScript = true

        let contentController = WKUserContentController()
        contentController.add(context.coordinator.bridge, name: "gachaBridge")
        let disableSelectionJS = """
        document.documentElement.style.webkitUserSelect = 'none';
        document.documentElement.style.webkitTouchCallout = 'none';
        document.addEventListener('contextmenu', function(e) { e.preventDefault(); }, { passive: false });
        document.addEventListener('selectstart', function(e) { e.preventDefault(); }, { passive: false });
        """
        contentController.addUserScript(
            WKUserScript(source: disableSelectionJS, injectionTime: .atDocumentEnd, forMainFrameOnly: false)
        )
        config.userContentController = contentController

        let webView = NoCalloutWebView(frame: .zero, configuration: config)
        webView.isOpaque = false
        webView.backgroundColor = .clear
        webView.allowsLinkPreview = false
        webView.isFindInteractionEnabled = false
        webView.scrollView.contentInsetAdjustmentBehavior = .never
        webView.scrollView.bounces = false
        webView.scrollView.bouncesZoom = false
        webView.scrollView.minimumZoomScale = 1.0
        webView.scrollView.maximumZoomScale = 1.0
        webView.scrollView.pinchGestureRecognizer?.isEnabled = false
        webView.scrollView.delegate = context.coordinator.navDelegate
        webView.navigationDelegate = context.coordinator.navDelegate

        context.coordinator.bridge.webView = webView
        context.coordinator.bridge.onCloseRequested = onCloseRequested
        context.coordinator.navDelegate.onLoadError = { err in
            onError(err)
        }
        context.coordinator.navDelegate.onLoadFinished = {
            onError(nil)
        }

        context.coordinator.lastReloadToken = reloadToken
        context.coordinator.lastURL = url
        context.coordinator.navDelegate.markNavigationStart()
        webView.load(URLRequest(url: url, cachePolicy: .reloadIgnoringLocalCacheData, timeoutInterval: 10))
        return webView
    }

    func updateUIView(_ uiView: WKWebView, context: Context) {
        if context.coordinator.lastReloadToken != reloadToken || context.coordinator.lastURL != url {
            context.coordinator.lastReloadToken = reloadToken
            context.coordinator.lastURL = url
            context.coordinator.navDelegate.markNavigationStart()
            uiView.load(URLRequest(url: url, cachePolicy: .reloadIgnoringLocalCacheData, timeoutInterval: 10))
        }
    }

    static func dismantleUIView(_ uiView: WKWebView, coordinator: ContainerCoordinator) {
        uiView.scrollView.delegate = nil
        uiView.configuration.userContentController.removeScriptMessageHandler(forName: "gachaBridge")
    }

    @MainActor
    final class ContainerCoordinator {
        let bridge: NativeBridgeHandler
        let navDelegate: HardenedWebViewCoordinator
        var lastReloadToken: UUID?
        var lastURL: URL?

        init(bridge: NativeBridgeHandler, navDelegate: HardenedWebViewCoordinator) {
            self.bridge = bridge
            self.navDelegate = navDelegate
        }
    }
}

struct WebViewFallbackView: View {
    let onDisableFallback: () -> Void

    var body: some View {
        GachaCard(title: "Fallback ke Jalur Lama Aktif") {
            VStack(alignment: .leading, spacing: 12) {
                Label("Feature Flag `webViewFallbackEnabled` aktif", systemImage: "arrow.uturn.backward.circle.fill")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(GachaTheme.primaryAccent)
                Text("Super app dapat mengalihkan trafik kembali ke mini program lama tanpa merilis ulang aplikasi.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                Button("Kembalikan ke WebView OCP", action: onDisableFallback)
                    .buttonStyle(.borderedProminent)
            }
        }
        .padding()
    }
}

struct WebViewApproachScreen: View {
    @Bindable var config: AppConfig
    let tokenManager: TokenManager
    let pinner: CertificatePinner
    let metrics: MetricsCollector

    @Environment(\.scenePhase) private var scenePhase
    @Environment(\.dismiss) private var dismiss
    @State private var reloadToken = UUID()
    @State private var loadErrorMessage: String?
    @State private var showOptionsSheet = false

    var body: some View {
        ZStack(alignment: .top) {
            if config.webViewFallbackEnabled {
                ZStack {
                    Color(.systemGroupedBackground).ignoresSafeArea()
                    WebViewFallbackView {
                        config.webViewFallbackEnabled = false
                    }
                    .padding(.top, 70)
                }
            } else {
                GachaWebViewContainer(
                    url: config.webViewEntryURL,
                    reloadToken: reloadToken,
                    tokenManager: tokenManager,
                    pinner: pinner,
                    metrics: metrics,
                    onError: { err in
                        loadErrorMessage = err
                    },
                    onCloseRequested: {
                        dismiss()
                    }
                )
                .ignoresSafeArea()

                if let loadErrorMessage {
                    VStack(spacing: 12) {
                        Image(systemName: "wifi.exclamationmark")
                            .font(.largeTitle)
                            .foregroundStyle(GachaTheme.dangerColor)
                        Text("Gagal Memuat WebView SPA")
                            .font(.headline)
                        Text(loadErrorMessage)
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                            .multilineTextAlignment(.center)
                        Button("Coba Muat Ulang") {
                            self.loadErrorMessage = nil
                            reloadToken = UUID()
                        }
                        .buttonStyle(.borderedProminent)
                    }
                    .padding(24)
                    .background(.ultraThickMaterial, in: RoundedRectangle(cornerRadius: 16))
                    .padding()
                    .padding(.top, 100)
                }
            }

            VillageTransparentNavHeader(
                title: "Livin' Village",
                onBack: { dismiss() },
                onMore: { showOptionsSheet = true }
            )

            if showOptionsSheet {
                CenteredModalCard(
                    title: "Pengaturan WebView",
                    subtitle: "Pendekatan A · Vue 3",
                    onClose: { showOptionsSheet = false }
                ) {
                    Toggle("Gunakan Tampilan Lama", isOn: $config.webViewFallbackEnabled)
                        .font(.subheadline.weight(.medium))
                        .padding(12)
                        .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 12))

                    VStack(alignment: .leading, spacing: 4) {
                        Text("Proteksi Aktif")
                            .font(.subheadline.weight(.semibold))
                        Text("Domain terverifikasi, sesi terisolasi, dan akses bridge terbatas.")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    }
                    .padding(12)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 12))

                    Button {
                        loadErrorMessage = nil
                        reloadToken = UUID()
                        showOptionsSheet = false
                    } label: {
                        Text("Muat Ulang Halaman")
                            .font(.subheadline.weight(.bold))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 6)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(GachaTheme.primaryAccent)
                }
            }
        }
        .toolbar(.hidden, for: .navigationBar)
        .onChange(of: scenePhase) { _, newPhase in
            if newPhase == .active && loadErrorMessage != nil {
                loadErrorMessage = nil
                reloadToken = UUID()
            }
        }
    }
}
