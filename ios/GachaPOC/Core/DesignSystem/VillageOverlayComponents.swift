import SwiftUI

/// Kanvas full-screen isometric yang menjaga koordinat pin tetap menempel tepat pada gedung di segala ukuran layar iPhone.
struct IsometricBackgroundMap<PinsContent: View>: View {
    let imageName: String
    var baseURL: URL? = nil
    @ViewBuilder let pinsContent: (_ mapSize: CGSize) -> PinsContent

    private let imageRatio: CGFloat = 852.0 / 1846.0

    private var remoteImageURL: URL? {
        let trimmed = imageName.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.hasPrefix("http://") || trimmed.hasPrefix("https://") {
            return URL(string: trimmed)
        }
        if trimmed.hasPrefix("/"), let baseURL {
            return URL(string: trimmed, relativeTo: baseURL)?.absoluteURL
        }
        return nil
    }

    var body: some View {
        GeometryReader { geo in
            let viewW = max(1, geo.size.width)
            let viewH = max(1, geo.size.height)
            let viewRatio = viewW / viewH

            let mapWidth: CGFloat = viewRatio > imageRatio ? viewW : viewH * imageRatio
            let mapHeight: CGFloat = viewRatio > imageRatio ? viewW / imageRatio : viewH
            let offsetX = (viewW - mapWidth) / 2.0
            let offsetY = (viewH - mapHeight) / 2.0

            ZStack(alignment: .topLeading) {
                Group {
                    if let remoteURL = remoteImageURL {
                        AsyncImage(url: remoteURL) { phase in
                            switch phase {
                            case let .success(image):
                                image.resizable()
                            default:
                                Image("bg-isometric").resizable()
                            }
                        }
                    } else {
                        Image(imageName)
                            .resizable()
                    }
                }
                .frame(width: mapWidth, height: mapHeight)
                .offset(x: offsetX, y: offsetY)

                ZStack(alignment: .topLeading) {
                    pinsContent(CGSize(width: mapWidth, height: mapHeight))
                }
                .frame(width: mapWidth, height: mapHeight)
                .offset(x: offsetX, y: offsetY)
            }
        }
        .ignoresSafeArea()
    }
}

/// Overlay header transparan di area langit atas (< Livin' Village dan pill ••• | ⓧ).
struct VillageTransparentNavHeader: View {
    let title: String
    let onBack: () -> Void
    let onMore: () -> Void

    var body: some View {
        HStack {
            Button(action: onBack) {
                HStack(spacing: 6) {
                    Image(systemName: "chevron.left")
                        .font(.system(size: 17, weight: .semibold))
                    Text(title)
                        .font(.system(size: 18, weight: .bold))
                }
                .foregroundStyle(.white)
                .shadow(color: .black.opacity(0.22), radius: 3, x: 0, y: 1)
            }

            Spacer()

            HStack(spacing: 0) {
                Button(action: onMore) {
                    Image(systemName: "ellipsis")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(.white)
                        .frame(width: 38, height: 30)
                }
                .accessibilityLabel("Menu opsi pendekatan")

                Rectangle()
                    .fill(.white.opacity(0.35))
                    .frame(width: 1, height: 16)

                Button(action: onBack) {
                    Image(systemName: "xmark.circle")
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(.white)
                        .frame(width: 38, height: 30)
                }
                .accessibilityLabel("Kembali ke Home")
            }
            .background(.black.opacity(0.22), in: Capsule())
            .overlay(Capsule().stroke(.white.opacity(0.35), lineWidth: 1))
        }
        .padding(.horizontal, 16)
        .padding(.top, 6)
    }
}

struct VillageCampaignBanner: View {
    let text: String

    var body: some View {
        Text(text)
            .font(.system(size: 13.5, weight: .bold, design: .rounded))
            .foregroundStyle(.white)
            .multilineTextAlignment(.center)
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .frame(maxWidth: 256)
            .background(
                LinearGradient(
                    colors: [
                        Color(red: 0.24, green: 0.64, blue: 1.0),
                        Color(red: 0.09, green: 0.46, blue: 0.91)
                    ],
                    startPoint: .top,
                    endPoint: .bottom
                ),
                in: RoundedRectangle(cornerRadius: 16, style: .continuous)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 16, style: .continuous)
                    .stroke(Color(red: 0.53, green: 0.80, blue: 1.0), lineWidth: 2)
            )
            .shadow(color: Color(red: 0.06, green: 0.30, blue: 0.63).opacity(0.35), radius: 10, x: 0, y: 6)
    }
}

struct VillageFloatingActionButton: View {
    enum Kind {
        case help
        case gift
    }

    let kind: Kind
    var badgeText: String? = nil
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            ZStack(alignment: .topTrailing) {
                ZStack {
                    RoundedRectangle(cornerRadius: 13, style: .continuous)
                        .fill(
                            LinearGradient(
                                colors: kind == .help
                                    ? [Color(red: 1.0, green: 0.84, blue: 0.27), Color(red: 0.90, green: 0.58, blue: 0.04)]
                                    : [Color(red: 0.31, green: 0.70, blue: 1.0), Color(red: 0.09, green: 0.45, blue: 0.90)],
                                startPoint: .top,
                                endPoint: .bottom
                            )
                        )
                        .frame(width: 46, height: 46)
                        .overlay(
                            RoundedRectangle(cornerRadius: 13, style: .continuous)
                                .stroke(.white.opacity(0.92), lineWidth: 2)
                        )
                        .shadow(color: .black.opacity(0.22), radius: 6, x: 0, y: 4)

                    if kind == .help {
                        Image(systemName: "questionmark")
                            .font(.system(size: 20, weight: .heavy))
                            .foregroundStyle(.white)
                    } else {
                        Image(systemName: "gift.fill")
                            .font(.system(size: 20, weight: .bold))
                            .foregroundStyle(.white)
                    }
                }

                if let badgeText {
                    Text(badgeText)
                        .font(.system(size: 10, weight: .bold).monospacedDigit())
                        .foregroundStyle(.white)
                        .padding(.horizontal, 5)
                        .padding(.vertical, 2)
                        .background(Color(.label), in: Capsule())
                        .overlay(Capsule().stroke(.white, lineWidth: 1.5))
                        .offset(x: 6, y: -6)
                }
            }
        }
        .accessibilityLabel(kind == .help ? "Panduan Kuis" : "Saldo dan Hadiah")
    }
}

struct VillageMapPinButton: View {
    let color: PinColor
    let completed: Bool
    let title: String
    let action: () -> Void

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var bobUp = false

    private var gradientColors: [Color] {
        switch color {
        case .blue:
            return [Color(red: 0.40, green: 0.72, blue: 1.0), Color(red: 0.10, green: 0.51, blue: 0.94)]
        case .orange:
            return [Color(red: 1.0, green: 0.82, blue: 0.40), Color(red: 0.96, green: 0.62, blue: 0.04)]
        case .red:
            return [Color(red: 1.0, green: 0.52, blue: 0.52), Color(red: 0.94, green: 0.27, blue: 0.27)]
        }
    }

    var body: some View {
        Button(action: action) {
            ZStack {
                Circle()
                    .fill(
                        RadialGradient(
                            colors: gradientColors,
                            center: .topLeading,
                            startRadius: 4,
                            endRadius: 40
                        )
                    )
                    .frame(width: 62, height: 62)
                    .overlay(Circle().stroke(.white, lineWidth: 3))
                    .shadow(color: .black.opacity(0.32), radius: 8, x: 0, y: 6)

                // Grafik clipboard & pensil mirip referensi Livin' Village
                ZStack {
                    Image(systemName: "list.clipboard.fill")
                        .font(.system(size: 26, weight: .semibold))
                        .foregroundStyle(.white)
                    Image(systemName: "pencil")
                        .font(.system(size: 16, weight: .black))
                        .foregroundStyle(color == .orange ? Color.blue : Color.yellow)
                        .offset(x: 9, y: 6)
                }

                if completed {
                    Image(systemName: "checkmark.circle.fill")
                        .font(.system(size: 20, weight: .bold))
                        .foregroundStyle(.green, .white)
                        .offset(x: 22, y: -22)
                }
            }
        }
        .offset(y: (!reduceMotion && bobUp) ? -5 : 0)
        .onAppear {
            guard !reduceMotion else { return }
            withAnimation(.easeInOut(duration: 1.8).repeatForever(autoreverses: true)) {
                bobUp = true
            }
        }
        .accessibilityLabel(title)
    }
}

/// Card modal terpusat dengan latar solid agar kontras dan keterbacaan tetap terjaga di atas peta.
struct CenteredModalCard<Content: View>: View {
    let title: String
    var subtitle: String? = nil
    let onClose: () -> Void
    @ViewBuilder let content: () -> Content

    var body: some View {
        ZStack {
            Color.black.opacity(0.45)
                .ignoresSafeArea()
                .onTapGesture(perform: onClose)

            VStack(alignment: .leading, spacing: 16) {
                HStack(alignment: .top, spacing: 12) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text(title)
                            .font(.title3.weight(.bold))
                            .foregroundStyle(.primary)
                        if let subtitle {
                            Text(subtitle)
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                        }
                    }
                    Spacer()
                    Button(action: onClose) {
                        Image(systemName: "xmark")
                            .font(.system(size: 13, weight: .bold))
                            .foregroundStyle(.secondary)
                            .frame(width: 32, height: 32)
                            .background(Color(.secondarySystemBackground), in: Circle())
                    }
                    .accessibilityLabel("Tutup")
                }

                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        content()
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                .scrollBounceBehavior(.basedOnSize)
            }
            .padding(20)
            .frame(maxWidth: 380)
            .frame(maxHeight: 560)
            .fixedSize(horizontal: false, vertical: true)
            .background(Color(.systemBackground), in: RoundedRectangle(cornerRadius: 22, style: .continuous))
            .shadow(color: .black.opacity(0.28), radius: 24, x: 0, y: 12)
            .padding(.horizontal, 20)
        }
    }
}

struct VillageQuizSheet: View {
    let quiz: VillageQuiz
    @Binding var selectedIndex: Int?
    let isSubmitting: Bool
    let submitResult: QuizSubmitResponse?
    let errorBanner: GachaAPIError?
    let hasPendingKey: Bool
    let onSubmit: () -> Void
    let onClose: () -> Void

    var body: some View {
        CenteredModalCard(
            title: quiz.buildingName,
            subtitle: quiz.title,
            onClose: onClose
        ) {
            Text("+\(quiz.rewardPoints) Poin")
                .font(.caption.weight(.bold).monospacedDigit())
                .padding(.horizontal, 10)
                .padding(.vertical, 4)
                .foregroundStyle(GachaTheme.primaryAccent)
                .background(GachaTheme.primaryAccent.opacity(0.12), in: Capsule())

            Text(quiz.question)
                .font(.body.weight(.medium))
                .fixedSize(horizontal: false, vertical: true)

            if quiz.completed {
                HStack(alignment: .top, spacing: 10) {
                    Image(systemName: "checkmark.circle.fill")
                        .font(.title3)
                        .foregroundStyle(GachaTheme.successColor)
                    VStack(alignment: .leading, spacing: 2) {
                        Text("Kuis selesai")
                            .font(.subheadline.weight(.bold))
                        Text("Hadiah +\(quiz.rewardPoints) poin sudah masuk ke saldo Anda.")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    }
                }
                .padding(12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(GachaTheme.successColor.opacity(0.10), in: RoundedRectangle(cornerRadius: 12))
            } else {
                VStack(spacing: 10) {
                    ForEach(Array(quiz.options.enumerated()), id: \.offset) { idx, option in
                        Button {
                            selectedIndex = idx
                        } label: {
                            HStack(spacing: 12) {
                                ZStack {
                                    Circle()
                                        .stroke(selectedIndex == idx ? GachaTheme.primaryAccent : Color(.tertiaryLabel), lineWidth: 2)
                                        .frame(width: 20, height: 20)
                                    if selectedIndex == idx {
                                        Circle()
                                            .fill(GachaTheme.primaryAccent)
                                            .frame(width: 10, height: 10)
                                    }
                                }
                                Text(option)
                                    .font(.subheadline)
                                    .foregroundStyle(.primary)
                                    .multilineTextAlignment(.leading)
                                Spacer()
                            }
                            .padding(12)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(
                                selectedIndex == idx
                                    ? GachaTheme.primaryAccent.opacity(0.10)
                                    : Color(.secondarySystemBackground),
                                in: RoundedRectangle(cornerRadius: 12)
                            )
                            .overlay(
                                RoundedRectangle(cornerRadius: 12)
                                    .stroke(selectedIndex == idx ? GachaTheme.primaryAccent : .clear, lineWidth: 1.5)
                            )
                        }
                    }
                }
            }

            if let errorBanner {
                NativeErrorBanner(
                    error: errorBanner,
                    hasPendingKey: hasPendingKey,
                    onRetry: onSubmit
                )
            }

            if let submitResult {
                VStack(alignment: .leading, spacing: 4) {
                    Text(submitResult.correct ? "Benar · +\(submitResult.rewardPoints) Poin" : "Belum tepat")
                        .font(.subheadline.weight(.bold))
                        .foregroundStyle(submitResult.correct ? GachaTheme.successColor : GachaTheme.dangerColor)
                    Text(submitResult.explanation)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                .padding(12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(
                    (submitResult.correct ? GachaTheme.successColor : GachaTheme.dangerColor).opacity(0.10),
                    in: RoundedRectangle(cornerRadius: 12)
                )
            }

            if !quiz.completed {
                Button {
                    onSubmit()
                } label: {
                    HStack {
                        if isSubmitting {
                            ProgressView().tint(.white)
                            Text("Mengirim...")
                        } else {
                            Text(hasPendingKey ? "Coba Lagi" : "Kirim Jawaban")
                        }
                    }
                    .font(.subheadline.weight(.bold))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                }
                .buttonStyle(.borderedProminent)
                .tint(GachaTheme.primaryAccent)
                .disabled(selectedIndex == nil || isSubmitting)
            }
        }
    }
}

// MARK: - Subpage Header & Ribbon (Cara Main & Reward Anda)

private let livinBlue = Color(red: 0.0, green: 0.353, blue: 0.886)

private struct VillageRibbonBanner: View {
    let title: String

    var body: some View {
        Text(title)
            .font(.system(size: 16, weight: .bold))
            .foregroundStyle(.white)
            .padding(.horizontal, 44)
            .padding(.vertical, 10)
            .background(
                LinearGradient(
                    colors: [
                        Color(red: 1.0, green: 0.72, blue: 0.20),
                        Color(red: 0.96, green: 0.58, blue: 0.07)
                    ],
                    startPoint: .top,
                    endPoint: .bottom
                ),
                in: RoundedRectangle(cornerRadius: 12, style: .continuous)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(Color(red: 1.0, green: 0.85, blue: 0.35), lineWidth: 1.5)
            )
            .shadow(color: .black.opacity(0.16), radius: 6, x: 0, y: 3)
    }
}

private struct VillageClipboardBadge: View {
    var size: CGFloat = 68

    var body: some View {
        ZStack {
            Circle()
                .fill(
                    LinearGradient(
                        colors: [Color(red: 0.56, green: 0.83, blue: 1.0), Color(red: 0.16, green: 0.53, blue: 0.94)],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )
                .frame(width: size, height: size)
                .overlay(Circle().stroke(.white, lineWidth: 3))
                .shadow(color: .black.opacity(0.18), radius: 8, x: 0, y: 4)

            Image(systemName: "list.clipboard.fill")
                .font(.system(size: size * 0.42, weight: .semibold))
                .foregroundStyle(.white)
                .overlay(alignment: .bottomTrailing) {
                    Image(systemName: "pencil.circle.fill")
                        .font(.system(size: size * 0.28))
                        .foregroundStyle(Color(red: 1.0, green: 0.78, blue: 0.15))
                        .background(Circle().fill(.white).padding(1))
                        .offset(x: 4, y: 4)
                }
        }
    }
}

private struct VillageSubpageShell<Content: View>: View {
    let title: String
    let showCenterBadge: Bool
    let imageAlignment: Alignment
    let onClose: () -> Void
    @ViewBuilder let content: () -> Content

    var body: some View {
        ZStack(alignment: .top) {
            Color.white.ignoresSafeArea()

            VStack(spacing: 0) {
                ZStack(alignment: .bottom) {
                    ZStack(alignment: .top) {
                        Image("bg-isometric")
                            .resizable()
                            .scaledToFill()
                            .frame(height: 176, alignment: imageAlignment)
                            .clipped()
                            .overlay(
                                LinearGradient(
                                    colors: [
                                        Color(red: 0.55, green: 0.80, blue: 0.96).opacity(0.85),
                                        Color.clear
                                    ],
                                    startPoint: .top,
                                    endPoint: .center
                                )
                            )

                        if showCenterBadge {
                            VillageClipboardBadge(size: 74)
                                .padding(.top, 56)
                        }

                        VillageTransparentNavHeader(
                            title: title,
                            onBack: onClose,
                            onMore: onClose
                        )
                        .padding(.top, 6)
                    }

                    Rectangle()
                        .fill(Color(red: 1.0, green: 0.76, blue: 0.16))
                        .frame(height: 5)

                    VillageRibbonBanner(title: title)
                        .offset(y: 18)
                }
                .padding(.bottom, 26)

                ScrollView(showsIndicators: false) {
                    VStack(spacing: 20) {
                        content()
                    }
                    .padding(.horizontal, 18)
                    .padding(.top, 6)
                    .padding(.bottom, 32)
                }
            }
        }
    }
}

struct VillageHelpSheet: View {
    let rules: [String]
    let probabilities: ProbabilitiesResponse?
    let onClose: () -> Void

    private let termsAndConditions: [String] = [
        "Pilih kuis yang akan Anda kerjakan.",
        "Pilih jawaban berdasarkan petunjuk yang ada di setiap pertanyaan dalam waktu yang telah ditentukan.",
        "Selesaikan kuis dan lihat skor yang Anda dapatkan.",
        "Dapatkan hadiah menarik bagi Anda yang berhasil menjawab di atas skor minimal kuis.",
        "Bank Mandiri dapat menarik kembali kuis sesuai dengan kebijakan Bank."
    ]

    var body: some View {
        VillageSubpageShell(
            title: "Cara Main",
            showCenterBadge: false,
            imageAlignment: .center,
            onClose: onClose
        ) {
            // Kartu biru 3 langkah bermain
            HStack(alignment: .top, spacing: 10) {
                VStack(spacing: 10) {
                    VillageClipboardBadge(size: 54)
                        .frame(height: 56)
                    Text("Pilih satu tema\nkuis")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(.white)
                        .multilineTextAlignment(.center)
                }
                .frame(maxWidth: .infinity)

                VStack(spacing: 10) {
                    HStack(spacing: 5) {
                        Image(systemName: "checkmark")
                            .font(.system(size: 11, weight: .heavy))
                        Text("Correct")
                            .font(.system(size: 13, weight: .bold))
                    }
                    .foregroundStyle(.white)
                    .padding(.horizontal, 14)
                    .padding(.vertical, 8)
                    .background(
                        LinearGradient(
                            colors: [Color(red: 0.49, green: 0.87, blue: 0.25), Color(red: 0.34, green: 0.73, blue: 0.14)],
                            startPoint: .top,
                            endPoint: .bottom
                        ),
                        in: Capsule()
                    )
                    .overlay(Capsule().stroke(.white.opacity(0.65), lineWidth: 1.5))
                    .frame(height: 56)

                    Text("Jawab semua\npertanyaannya")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(.white)
                        .multilineTextAlignment(.center)
                }
                .frame(maxWidth: .infinity)

                VStack(spacing: 10) {
                    ZStack {
                        RoundedRectangle(cornerRadius: 12, style: .continuous)
                            .fill(
                                LinearGradient(
                                    colors: [Color(red: 0.26, green: 0.67, blue: 1.0), Color(red: 0.08, green: 0.43, blue: 0.90)],
                                    startPoint: .top,
                                    endPoint: .bottom
                                )
                            )
                            .frame(width: 48, height: 48)
                            .overlay(
                                RoundedRectangle(cornerRadius: 12, style: .continuous)
                                    .stroke(Color(red: 1.0, green: 0.84, blue: 0.25), lineWidth: 2)
                            )
                        Image(systemName: "gift.fill")
                            .font(.system(size: 24, weight: .bold))
                            .foregroundStyle(Color(red: 1.0, green: 0.84, blue: 0.25))
                    }
                    .frame(height: 56)

                    Text("Menangkan\nreward!")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(.white)
                        .multilineTextAlignment(.center)
                }
                .frame(maxWidth: .infinity)
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 20)
            .background(livinBlue, in: RoundedRectangle(cornerRadius: 14, style: .continuous))

            // Syarat dan Ketentuan
            VStack(spacing: 14) {
                Text("Syarat dan Ketentuan")
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(Color(red: 0.13, green: 0.15, blue: 0.18))
                    .frame(maxWidth: .infinity, alignment: .center)

                VStack(alignment: .leading, spacing: 8) {
                    ForEach(Array(termsAndConditions.enumerated()), id: \.offset) { idx, item in
                        HStack(alignment: .top, spacing: 6) {
                            Text("\(idx + 1).")
                                .font(.system(size: 14))
                                .foregroundStyle(Color(red: 0.20, green: 0.22, blue: 0.26))
                            Text(item)
                                .font(.system(size: 14))
                                .foregroundStyle(Color(red: 0.20, green: 0.22, blue: 0.26))
                                .fixedSize(horizontal: false, vertical: true)
                        }
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .padding(.top, 4)
        }
    }
}

struct VillageRewardsSheet: View {
    let approach: ClientApproach
    let balance: Int64
    let completedCount: Int
    let totalQuizzes: Int
    var quizzes: [VillageQuiz] = []
    let probabilities: ProbabilitiesResponse?
    let history: [HistoryItem]
    let lastDraw: DrawResponse?
    let isDrawing: Bool
    let isAnimating: Bool
    let errorBanner: GachaAPIError?
    let hasPendingDrawKey: Bool
    let metrics: MetricsCollector
    let onDraw: () -> Void
    let onAnimationComplete: (DrawResponse) -> Void
    let onClose: () -> Void

    private var completedQuizzes: [VillageQuiz] {
        quizzes.filter(\.completed)
    }

    var body: some View {
        VillageSubpageShell(
            title: "Reward Anda",
            showCenterBadge: true,
            imageAlignment: .top,
            onClose: onClose
        ) {
            if let errorBanner {
                NativeErrorBanner(
                    error: errorBanner,
                    hasPendingKey: hasPendingDrawKey,
                    onRetry: onDraw
                )
            }

            if completedQuizzes.isEmpty && history.isEmpty {
                VStack(spacing: 10) {
                    LivinCoinBadge()
                    Text("Belum ada reward")
                        .font(.system(size: 16, weight: .bold))
                        .foregroundStyle(.white)
                    Text("Selesaikan kuis di Livin' Village untuk mendapatkan Livin'poin.")
                        .font(.system(size: 13))
                        .foregroundStyle(.white.opacity(0.88))
                        .multilineTextAlignment(.center)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 22)
                .padding(.horizontal, 16)
                .background(livinBlue, in: RoundedRectangle(cornerRadius: 14, style: .continuous))
            } else {
                VStack(spacing: 12) {
                    ForEach(history) { item in
                        RewardEntryCard(
                            title: "\(item.rewardPoints) Livin'poin",
                            subtitle: "Didapat pada \(Self.formatWIB(item.createdAt))"
                        )
                    }

                    ForEach(completedQuizzes) { quiz in
                        RewardEntryCard(
                            title: "\(quiz.rewardPoints) Livin'poin",
                            subtitle: "\(quiz.buildingName) · \(quiz.title)"
                        )
                    }
                }
            }

            if isAnimating || lastDraw != nil {
                GachaBoxAnimationView(
                    approach: approach,
                    isAnimating: isAnimating,
                    lastDraw: lastDraw,
                    metrics: metrics,
                    onCompleted: onAnimationComplete
                )
            }

            // Aksi tukar kotak hadiah tetap tersedia di bawah daftar reward
            let cost = probabilities?.drawCost ?? 100
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Total Saldo")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("\(balance) Livin'poin")
                        .font(.system(size: 16, weight: .bold, design: .rounded))
                        .foregroundStyle(.primary)
                }
                Spacer()
                Button(action: onDraw) {
                    Text(isDrawing ? "Membuka..." : "Kotak Hadiah (\(cost) Poin)")
                        .font(.system(size: 13, weight: .bold))
                }
                .buttonStyle(.borderedProminent)
                .tint(livinBlue)
                .disabled(isDrawing || isAnimating || (balance < cost && !hasPendingDrawKey))
            }
            .padding(14)
            .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
        }
    }

    private static func formatWIB(_ iso: String) -> String {
        let isoParser = ISO8601DateFormatter()
        isoParser.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let fallbackParser = ISO8601DateFormatter()
        guard let date = isoParser.date(from: iso) ?? fallbackParser.date(from: iso) else {
            return iso
        }
        let fmt = DateFormatter()
        fmt.locale = Locale(identifier: "id_ID")
        fmt.timeZone = TimeZone(identifier: "Asia/Jakarta")
        fmt.dateFormat = "dd MMM yyyy HH:mm 'WIB'"
        return fmt.string(from: date)
    }
}

private struct LivinCoinBadge: View {
    var body: some View {
        ZStack {
            Circle()
                .fill(
                    LinearGradient(
                        colors: [
                            Color(red: 1.0, green: 0.88, blue: 0.30),
                            Color(red: 0.95, green: 0.64, blue: 0.08)
                        ],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )
                .frame(width: 44, height: 44)
                .overlay(
                    Circle()
                        .stroke(Color(red: 1.0, green: 0.95, blue: 0.65), lineWidth: 2)
                )
            VStack(spacing: -2) {
                Text("livin'")
                    .font(.system(size: 10, weight: .heavy, design: .rounded))
                Text("poin")
                    .font(.system(size: 8, weight: .bold, design: .rounded))
            }
            .foregroundStyle(.white)
        }
    }
}

private struct RewardEntryCard: View {
    let title: String
    let subtitle: String

    var body: some View {
        HStack(spacing: 14) {
            LivinCoinBadge()
            VStack(alignment: .leading, spacing: 4) {
                Text(title)
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(.white)
                Text(subtitle)
                    .font(.system(size: 12))
                    .foregroundStyle(.white.opacity(0.9))
            }
            Spacer()
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 16)
        .background(livinBlue, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }
}
