import SwiftUI

struct GachaBoxAnimationView: View {
    let approach: ClientApproach
    let isAnimating: Bool
    let lastDraw: DrawResponse?
    let metrics: MetricsCollector
    let onCompleted: (DrawResponse) -> Void

    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var animationStart: Date?

    private let duration: TimeInterval = 1.1

    var body: some View {
        VStack(spacing: 12) {
            TimelineView(.animation(minimumInterval: nil, paused: !isAnimating || reduceMotion)) { timeline in
                let progress: Double = {
                    guard isAnimating, let animationStart else {
                        return lastDraw != nil ? 1.0 : 0.0
                    }
                    if reduceMotion { return 1.0 }
                    return min(1.0, timeline.date.timeIntervalSince(animationStart) / duration)
                }()

                Canvas { ctx, size in
                    let shakeX: CGFloat = (isAnimating && !reduceMotion && progress < 0.65)
                        ? sin(progress * .pi * 10) * 6
                        : 0
                    let lidPhase = progress < 0.65 ? 0 : (progress - 0.65) / 0.35

                    let center = CGPoint(x: size.width / 2 + shakeX, y: size.height / 2 - 4)

                    // Aura glow
                    let auraRadius = 46 + lidPhase * 16
                    let auraRect = CGRect(
                        x: center.x - auraRadius,
                        y: center.y - auraRadius,
                        width: auraRadius * 2,
                        height: auraRadius * 2
                    )
                    ctx.fill(
                        Path(ellipseIn: auraRect),
                        with: .color(GachaTheme.primaryAccent.opacity(lidPhase > 0.7 ? 0.24 : 0.10))
                    )

                    // Box body
                    let bodyRect = CGRect(x: center.x - 39, y: center.y - 24, width: 78, height: 60)
                    ctx.fill(Path(roundedRect: bodyRect, cornerRadius: 8), with: .color(GachaTheme.primaryAccent))

                    // Ribbon
                    let ribbonRect = CGRect(x: center.x - 7, y: center.y - 24, width: 14, height: 60)
                    ctx.fill(Path(ribbonRect), with: .color(.yellow))

                    // Box lid
                    let lidY = center.y - 36 - CGFloat(lidPhase) * 14
                    let lidRect = CGRect(x: center.x - 43, y: lidY, width: 86, height: 16)
                    ctx.fill(Path(roundedRect: lidRect, cornerRadius: 5), with: .color(.blue))
                }
                .frame(height: 160)
                .frame(maxWidth: .infinity)
                .background(GachaTheme.subtleBackground, in: RoundedRectangle(cornerRadius: 12))
            }

            if let lastDraw, !isAnimating {
                HStack(spacing: 8) {
                    RarityBadge(rarity: lastDraw.reward.rarity)
                    Text("Dapat \(lastDraw.reward.name) (+\(lastDraw.reward.rewardPoints) poin)")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(GachaTheme.successColor)
                    if lastDraw.idempotencyReplayed {
                        Text("· Replay")
                            .font(.caption.weight(.bold))
                            .foregroundStyle(.secondary)
                    }
                }
                .padding(.vertical, 8)
                .padding(.horizontal, 12)
                .frame(maxWidth: .infinity)
                .background(GachaTheme.successColor.opacity(0.12), in: RoundedRectangle(cornerRadius: 10))
            }
        }
        .onChange(of: isAnimating) { _, active in
            guard active, let draw = lastDraw else { return }
            if reduceMotion {
                onCompleted(draw)
                return
            }
            animationStart = Date()
            metrics.startDisplayLinkFPS(for: approach)
            Task { @MainActor in
                try? await Task.sleep(for: .seconds(duration))
                metrics.finishDisplayLinkFPS()
                animationStart = nil
                onCompleted(draw)
            }
        }
    }
}
