import SwiftUI

struct SDUIGachaScreen: View {
    @Bindable var viewModel: SDUIGachaViewModel
    @Bindable var nativeFallbackVM: NativeGachaViewModel
    let metrics: MetricsCollector
    @Bindable var config: AppConfig

    @Environment(\.dismiss) private var dismiss
    @State private var showOptionsSheet = false

    var body: some View {
        ZStack(alignment: .top) {
            if let reason = viewModel.fallbackReason {
                SDUIFallbackView(
                    reason: reason.reasonText,
                    nativeVM: nativeFallbackVM,
                    metrics: metrics,
                    onResetSchema: {
                        config.simulateUnsupportedSDUISchema = false
                        Task { await viewModel.loadAll(resetScreen: true) }
                    },
                    onBack: { dismiss() },
                    onMore: { showOptionsSheet = true }
                )
            } else if let screen = viewModel.validatedScreen {
                SDUIVillageRendererView(
                    screen: screen,
                    baseURL: config.baseURL,
                    viewModel: viewModel,
                    onBack: { dismiss() },
                    onMore: { showOptionsSheet = true }
                )
            } else {
                IsometricBackgroundMap(imageName: "bg-isometric") { _ in EmptyView() }
                VStack {
                    VillageTransparentNavHeader(
                        title: "Livin' Village",
                        onBack: { dismiss() },
                        onMore: { showOptionsSheet = true }
                    )
                    Spacer()
                    if let err = viewModel.errorBanner {
                        NativeErrorBanner(
                            error: err,
                            hasPendingKey: false,
                            onRetry: { Task { await viewModel.loadAll(resetScreen: true) } }
                        )
                        .padding()
                    } else {
                        ProgressView("Merangkai layar SDUI dari JSON server...")
                            .padding(16)
                            .background(.ultraThickMaterial, in: RoundedRectangle(cornerRadius: 14))
                    }
                    Spacer()
                }
            }

            if let route = viewModel.activeSheet {
                switch route {
                case let .quiz(quizId):
                    if let quiz = viewModel.quiz(for: quizId) {
                        VillageQuizSheet(
                            quiz: quiz,
                            selectedIndex: $viewModel.selectedQuizOptionIndex,
                            isSubmitting: viewModel.isSubmittingQuiz,
                            submitResult: viewModel.lastQuizResult,
                            errorBanner: viewModel.errorBanner,
                            hasPendingKey: viewModel.pendingQuizIdempotencyKey != nil,
                            onSubmit: { Task { await viewModel.submitQuiz(quizId: quiz.id) } },
                            onClose: { viewModel.closeSheet() }
                        )
                    }
                case .help:
                    VillageHelpSheet(
                        rules: viewModel.villageState?.rules ?? [],
                        probabilities: viewModel.probabilities,
                        onClose: { viewModel.closeSheet() }
                    )
                case .rewards:
                    VillageRewardsSheet(
                        approach: .sdui,
                        balance: viewModel.balance,
                        completedCount: viewModel.villageState?.completedCount ?? 0,
                        totalQuizzes: viewModel.villageState?.totalQuizzes ?? 3,
                        quizzes: viewModel.villageState?.quizzes ?? [],
                        probabilities: viewModel.probabilities,
                        history: viewModel.history,
                        lastDraw: viewModel.lastDraw,
                        isDrawing: viewModel.drawing,
                        isAnimating: viewModel.animating,
                        errorBanner: viewModel.errorBanner,
                        hasPendingDrawKey: viewModel.pendingIdempotencyKey != nil,
                        metrics: metrics,
                        onDraw: { Task { await viewModel.executeDraw() } },
                        onAnimationComplete: { _ in viewModel.completeAnimation() },
                        onClose: { viewModel.closeSheet() }
                    )
                }
            }

            if showOptionsSheet {
                CenteredModalCard(
                    title: "Pengaturan SDUI",
                    subtitle: "Pendekatan C · Susunan dari Server",
                    onClose: { showOptionsSheet = false }
                ) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("Varian Tampilan")
                            .font(.subheadline.weight(.semibold))
                        Picker("Varian Tampilan", selection: $config.sduiVariant) {
                            Text("Default").tag("default")
                            Text("Promo").tag("promo")
                            Text("Ringkas").tag("compact")
                        }
                        .pickerStyle(.segmented)
                        .onChange(of: config.sduiVariant) { _, _ in
                            Task { await viewModel.loadAll(resetScreen: true) }
                        }
                    }

                    Toggle("Simulasi Versi Skema Baru (Fallback)", isOn: $config.simulateUnsupportedSDUISchema)
                        .font(.subheadline.weight(.medium))
                        .padding(12)
                        .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 12))
                        .onChange(of: config.simulateUnsupportedSDUISchema) { _, _ in
                            Task { await viewModel.loadAll(resetScreen: true) }
                        }

                    if let screen = viewModel.validatedScreen {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("Komponen Aktif (\(screen.nodes.count))")
                                .font(.subheadline.weight(.semibold))
                            ForEach(screen.nodes) { node in
                                HStack {
                                    Text(node.id)
                                        .font(.caption.monospaced())
                                    Spacer()
                                    Text(node.type.rawValue)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                        .padding(12)
                        .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 12))
                    }
                }
            }
        }
        .toolbar(.hidden, for: .navigationBar)
        .task {
            await viewModel.loadAll(resetScreen: true)
        }
    }
}

/// Merangkai komponen isometric full-screen (banner, tombol floating, dan pin kuis) langsung dari node JSON SDUI yang sudah tervalidasi.
struct SDUIVillageRendererView: View {
    let screen: ValidatedSDUIScreen
    var baseURL: URL? = nil
    @Bindable var viewModel: SDUIGachaViewModel
    let onBack: () -> Void
    let onMore: () -> Void

    private var mapPins: [ValidatedSDUINode] {
        screen.nodes.filter { $0.type == .mapPin }
    }

    private var bannerNode: ValidatedSDUINode? {
        screen.nodes.first { $0.type == .floatingBanner }
    }

    private var actionNodes: [ValidatedSDUINode] {
        screen.nodes.filter { $0.type == .floatingAction }
    }

    var body: some View {
        ZStack(alignment: .top) {
            IsometricBackgroundMap(imageName: screen.backgroundAsset ?? "bg-isometric", baseURL: baseURL) { mapSize in
                ForEach(mapPins) { node in
                    let xRatio = node.props["x_ratio"]?.doubleValue ?? 0.5
                    let yRatio = node.props["y_ratio"]?.doubleValue ?? 0.5
                    let rawColor = node.props["pin_color"]?.stringValue ?? "blue"
                    let color = PinColor(rawValue: rawColor) ?? .blue
                    let completed = node.props["completed"]?.boolValue ?? false
                    let title = node.props["title"]?.stringValue ?? "Kuis"

                    VillageMapPinButton(
                        color: color,
                        completed: completed,
                        title: title
                    ) {
                        Task { await viewModel.dispatch(node: node) }
                    }
                    .position(
                        x: mapSize.width * xRatio,
                        y: mapSize.height * yRatio
                    )
                }
            }

            VStack(spacing: 10) {
                VillageTransparentNavHeader(
                    title: screen.navigationTitle ?? "Livin' Village",
                    onBack: onBack,
                    onMore: onMore
                )

                ZStack(alignment: .topTrailing) {
                    if let bannerNode {
                        HStack {
                            Spacer()
                            VillageCampaignBanner(
                                text: bannerNode.props["text"]?.stringValue ?? "Mainkan kuis Livin' Village"
                            )
                            Spacer()
                        }
                    }

                    VStack(spacing: 10) {
                        ForEach(actionNodes) { actionNode in
                            let isGift = actionNode.props["icon"]?.stringValue == "gift"
                            let badge = isGift
                                ? "\(viewModel.villageState?.completedCount ?? 0)/\(viewModel.villageState?.totalQuizzes ?? 3)"
                                : nil
                            VillageFloatingActionButton(
                                kind: isGift ? .gift : .help,
                                badgeText: badge
                            ) {
                                Task { await viewModel.dispatch(node: actionNode) }
                            }
                        }
                    }
                    .padding(.trailing, 14)
                }
                .padding(.top, 2)

                Spacer()
            }
        }
    }
}

struct SDUIFallbackView: View {
    let reason: String
    @Bindable var nativeVM: NativeGachaViewModel
    let metrics: MetricsCollector
    let onResetSchema: () -> Void
    let onBack: () -> Void
    let onMore: () -> Void

    var body: some View {
        ZStack(alignment: .top) {
            IsometricBackgroundMap(imageName: "bg-isometric") { _ in EmptyView() }

            VStack(spacing: 14) {
                VillageTransparentNavHeader(
                    title: "Livin' Village (Fallback)",
                    onBack: onBack,
                    onMore: onMore
                )

                GachaCard(title: "SDUI Fallback Aktif") {
                    VStack(alignment: .leading, spacing: 10) {
                        Text(reason)
                            .font(.footnote)
                            .foregroundStyle(GachaTheme.dangerColor)
                        Text("Karena schema_version tidak dikenali, app fallback ke layar bawaan yang aman.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        Button("Kembali ke Schema v1", action: onResetSchema)
                            .buttonStyle(.borderedProminent)
                    }
                }
                .padding(.horizontal, 16)

                NativeBalanceCard(
                    balance: nativeVM.balance,
                    isLoading: nativeVM.uiState == .loading,
                    onRefresh: { Task { await nativeVM.loadAll() } }
                )
                .padding(.horizontal, 16)

                Spacer()
            }
        }
        .task {
            if nativeVM.villageState == nil {
                await nativeVM.loadAll()
            }
        }
    }
}
