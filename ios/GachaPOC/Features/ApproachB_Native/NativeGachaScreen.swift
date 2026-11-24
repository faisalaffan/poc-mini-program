import SwiftUI

struct NativeGachaScreen: View {
    @Bindable var viewModel: NativeGachaViewModel
    let metrics: MetricsCollector

    @Environment(\.dismiss) private var dismiss
    @State private var showMoreSheet = false

    var body: some View {
        ZStack(alignment: .top) {
            IsometricBackgroundMap(imageName: "bg-isometric") { mapSize in
                if let quizzes = viewModel.villageState?.quizzes {
                    ForEach(quizzes) { quiz in
                        VillageMapPinButton(
                            color: quiz.pinColor,
                            completed: quiz.completed,
                            title: "\(quiz.buildingName): \(quiz.title)"
                        ) {
                            viewModel.openQuiz(quiz)
                        }
                        .position(
                            x: mapSize.width * quiz.xRatio,
                            y: mapSize.height * quiz.yRatio
                        )
                    }
                }
            }

            // Transparent top header + floating campaign banner + floating ? / gift buttons
            VStack(spacing: 10) {
                VillageTransparentNavHeader(
                    title: "Livin' Village",
                    onBack: { dismiss() },
                    onMore: { showMoreSheet = true }
                )

                ZStack(alignment: .topTrailing) {
                    HStack {
                        Spacer()
                        VillageCampaignBanner(
                            text: viewModel.villageState?.campaignBanner
                                ?? "Mainkan kuis Livin' Village & dapatkan hadiah menarik pada 1 Jun 2026 - 31 Dec 2026"
                        )
                        Spacer()
                    }

                    VStack(spacing: 10) {
                        VillageFloatingActionButton(kind: .help) {
                            viewModel.openHelp()
                        }
                        VillageFloatingActionButton(
                            kind: .gift,
                            badgeText: "\(viewModel.villageState?.completedCount ?? 0)/\(viewModel.villageState?.totalQuizzes ?? 3)"
                        ) {
                            viewModel.openRewards()
                        }
                    }
                    .padding(.trailing, 14)
                }
                .padding(.top, 2)

                if let err = viewModel.errorBanner, viewModel.activeSheet == nil {
                    Text(err.localizedDescription)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.white)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 8)
                        .background(GachaTheme.dangerColor.opacity(0.92), in: RoundedRectangle(cornerRadius: 10))
                        .padding(.horizontal, 20)
                }

                Spacer()
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
                    let isAnimating: Bool = {
                        if case .animating = viewModel.uiState { return true }
                        return false
                    }()
                    VillageRewardsSheet(
                        approach: .native,
                        balance: viewModel.balance,
                        completedCount: viewModel.villageState?.completedCount ?? 0,
                        totalQuizzes: viewModel.villageState?.totalQuizzes ?? 3,
                        quizzes: viewModel.villageState?.quizzes ?? [],
                        probabilities: viewModel.probabilities,
                        history: viewModel.history,
                        lastDraw: viewModel.lastDraw,
                        isDrawing: viewModel.uiState == .drawing,
                        isAnimating: isAnimating,
                        errorBanner: viewModel.errorBanner,
                        hasPendingDrawKey: viewModel.pendingIdempotencyKey != nil,
                        metrics: metrics,
                        onDraw: { Task { await viewModel.executeDraw() } },
                        onAnimationComplete: { res in viewModel.completeAnimation(for: res) },
                        onClose: { viewModel.closeSheet() }
                    )
                }
            }

            if showMoreSheet {
                CenteredModalCard(
                    title: "Pengaturan Native",
                    subtitle: "Pendekatan B · SwiftUI",
                    onClose: { showMoreSheet = false }
                ) {
                    Text("Layar dirender langsung dengan komponen native, token tersimpan di Keychain, dan koneksi diproteksi certificate pinning.")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                        .padding(12)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 12))

                    Button {
                        showMoreSheet = false
                        Task { await viewModel.loadAll() }
                    } label: {
                        Text("Segarkan Data")
                            .font(.subheadline.weight(.bold))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 6)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(GachaTheme.primaryAccent)

                    Button {
                        showMoreSheet = false
                        viewModel.openRewards()
                    } label: {
                        Text("Lihat Saldo & Hadiah")
                            .font(.subheadline.weight(.semibold))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 6)
                    }
                    .buttonStyle(.bordered)
                }
            }
        }
        .toolbar(.hidden, for: .navigationBar)
        .task {
            await viewModel.loadAll()
        }
    }
}

struct NativeBalanceCard: View {
    let balance: Int64
    let isLoading: Bool
    let onRefresh: () -> Void

    var body: some View {
        GachaCard(title: "Saldo Poin Anda (F1)") {
            HStack(alignment: .center) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(isLoading ? "Memuat..." : "\(balance) Poin")
                        .font(.system(size: 30, weight: .heavy, design: .rounded))
                    Text("Token tersimpan di iOS Keychain · Certificate Pinning aktif")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Segarkan", action: onRefresh)
                    .buttonStyle(.bordered)
                    .disabled(isLoading)
            }
        }
    }
}

struct NativeProbabilityCard: View {
    let probabilities: ProbabilitiesResponse

    var body: some View {
        GachaCard(title: "Daftar Probabilitas Hadiah (F2)") {
            Text("Tabel aktif: \(probabilities.tableVersion) · Biaya \(probabilities.drawCost) poin")
                .font(.caption)
                .foregroundStyle(.secondary)

            ForEach(probabilities.tiers) { tier in
                HStack {
                    RarityBadge(rarity: tier.rarity)
                    Text(tier.name)
                        .font(.subheadline)
                    Spacer()
                    Text("+\(tier.rewardPoints)")
                        .font(.subheadline.weight(.semibold))
                        .monospacedDigit()
                    Text(String(format: "%.1f%%", tier.probabilityPercent))
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .monospacedDigit()
                        .frame(width: 52, alignment: .trailing)
                }
                .padding(.vertical, 4)
            }
        }
    }
}

struct NativeHistoryCard: View {
    let items: [HistoryItem]

    var body: some View {
        GachaCard(title: "Riwayat Hadiah Terakhir (F5)") {
            if items.isEmpty {
                Text("Belum ada riwayat penukaran kotak hadiah.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            } else {
                ForEach(items) { item in
                    HStack {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(item.rewardName)
                                .font(.subheadline.weight(.semibold))
                            Text("\(item.tableVersion) · \(item.clientApproach.rawValue)")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        RarityBadge(rarity: item.rarity)
                        Text("+\(item.rewardPoints)")
                            .font(.subheadline.weight(.bold))
                            .monospacedDigit()
                    }
                    .padding(.vertical, 4)
                }
            }
        }
    }
}

struct NativeErrorBanner: View {
    let error: GachaAPIError
    let hasPendingKey: Bool
    let onRetry: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(error.errorCode)
                .font(.caption.weight(.bold))
            Text(error.localizedDescription)
                .font(.subheadline)
            if error.isRetryable || hasPendingKey {
                Button(hasPendingKey ? "Coba Lagi (Idempotency Key Sama)" : "Muat Ulang", action: onRetry)
                    .buttonStyle(.bordered)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(14)
        .foregroundStyle(GachaTheme.dangerColor)
        .background(GachaTheme.dangerColor.opacity(0.10), in: RoundedRectangle(cornerRadius: 12))
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(GachaTheme.dangerColor.opacity(0.35), lineWidth: 1)
        )
    }
}
