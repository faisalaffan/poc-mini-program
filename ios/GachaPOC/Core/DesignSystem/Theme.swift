import SwiftUI

enum GachaTheme {
    static let pageBackground = Color(.systemGroupedBackground)
    static let cardBackground = Color(.secondarySystemGroupedBackground)
    static let subtleBackground = Color(.tertiarySystemGroupedBackground)
    static let primaryAccent = Color(red: 0.06, green: 0.34, blue: 0.70)
    static let successColor = Color(red: 0.01, green: 0.48, blue: 0.28)
    static let dangerColor = Color(red: 0.71, green: 0.14, blue: 0.09)

    static func rarityColor(_ rarity: Rarity) -> Color {
        switch rarity {
        case .common: return Color(.secondaryLabel)
        case .uncommon: return successColor
        case .rare: return primaryAccent
        case .epic: return Color(red: 0.41, green: 0.25, blue: 0.78)
        case .jackpot: return Color(red: 0.71, green: 0.28, blue: 0.03)
        }
    }
}

struct GachaCard<Content: View>: View {
    let title: String?
    @ViewBuilder let content: Content

    init(title: String? = nil, @ViewBuilder content: () -> Content) {
        self.title = title
        self.content = content()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let title {
                Text(title)
                    .font(.headline)
                    .foregroundStyle(.primary)
            }
            content
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(GachaTheme.cardBackground, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .stroke(Color(.separator).opacity(0.35), lineWidth: 1)
        )
    }
}

struct RarityBadge: View {
    let rarity: Rarity

    var body: some View {
        Text(rarity.rawValue)
            .font(.caption2.weight(.bold))
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .foregroundStyle(GachaTheme.rarityColor(rarity))
            .overlay(
                Capsule().stroke(GachaTheme.rarityColor(rarity), lineWidth: 1)
            )
    }
}
