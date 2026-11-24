import SwiftUI

enum POCRoute: String, Hashable, CaseIterable, Identifiable {
    case webview
    case native
    case sdui
    case lab

    var id: String { rawValue }

    var title: String {
        switch self {
        case .webview: return "WebView"
        case .native: return "Native"
        case .sdui: return "SDUI"
        case .lab: return "Lab"
        }
    }

    var subtitle: String {
        switch self {
        case .webview: return "Vue 3 SPA"
        case .native: return "SwiftUI"
        case .sdui: return "Layout JSON"
        case .lab: return "Metrik & Uji"
        }
    }

    var systemImage: String {
        switch self {
        case .webview: return "globe"
        case .native: return "iphone"
        case .sdui: return "square.stack.3d.up"
        case .lab: return "gauge.with.dots.needle.67percent"
        }
    }
}

struct RootView: View {
    let container: AppContainer
    @State private var path: [POCRoute] = []

    var body: some View {
        NavigationStack(path: $path) {
            HomeLauncherView(config: container.config)
                .navigationDestination(for: POCRoute.self) { route in
                    switch route {
                    case .webview:
                        WebViewApproachScreen(
                            config: container.config,
                            tokenManager: container.tokenManager,
                            pinner: container.pinner,
                            metrics: container.metrics
                        )
                    case .native:
                        NativeGachaScreen(
                            viewModel: container.nativeVM,
                            metrics: container.metrics
                        )
                    case .sdui:
                        SDUIGachaScreen(
                            viewModel: container.sduiVM,
                            nativeFallbackVM: container.nativeVM,
                            metrics: container.metrics,
                            config: container.config
                        )
                    case .lab:
                        BenchmarkLabScreen(
                            config: container.config,
                            apiClient: container.apiClient,
                            tokenManager: container.tokenManager,
                            metrics: container.metrics,
                            onDataChanged: {
                                Task {
                                    await container.nativeVM.loadAll()
                                    await container.sduiVM.loadAll()
                                }
                            }
                        )
                    }
                }
        }
    }
}

private struct HomeLauncherView: View {
    let config: AppConfig

    private let columns = [
        GridItem(.flexible(), spacing: 20),
        GridItem(.flexible(), spacing: 20)
    ]

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            VStack(spacing: 32) {
                VStack(spacing: 6) {
                    Text("Livin' Village")
                        .font(.system(size: 32, weight: .bold, design: .rounded))
                    Text("Pilih metode tampilan")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                LazyVGrid(columns: columns, spacing: 24) {
                    ForEach(POCRoute.allCases) { route in
                        NavigationLink(value: route) {
                            HomeCircleButton(route: route)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal, 12)
            }

            Spacer()

            HStack(spacing: 8) {
                Circle()
                    .fill(GachaTheme.successColor)
                    .frame(width: 7, height: 7)
                Text(config.baseURL.host ?? config.baseURL.absoluteString)
                    .font(.caption.monospaced())
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
            .background(Color(.secondarySystemGroupedBackground), in: Capsule())
            .padding(.bottom, 16)
        }
        .padding(.horizontal, 20)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(GachaTheme.pageBackground)
        .toolbar(.hidden, for: .navigationBar)
    }
}

private struct HomeCircleButton: View {
    let route: POCRoute

    private var isLab: Bool { route == .lab }

    var body: some View {
        VStack(spacing: 10) {
            ZStack {
                Circle()
                    .fill(isLab ? Color(.tertiarySystemGroupedBackground) : GachaTheme.primaryAccent)
                    .frame(width: 88, height: 88)
                    .overlay(
                        Circle().stroke(isLab ? Color(.separator) : .clear, lineWidth: 1)
                    )
                    .shadow(color: .black.opacity(isLab ? 0.04 : 0.12), radius: 8, x: 0, y: 4)

                Image(systemName: route.systemImage)
                    .font(.system(size: 30, weight: .semibold))
                    .foregroundStyle(isLab ? GachaTheme.primaryAccent : .white)
            }

            VStack(spacing: 2) {
                Text(route.title)
                    .font(.headline)
                    .foregroundStyle(.primary)
                Text(route.subtitle)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 6)
        .contentShape(Rectangle())
    }
}
