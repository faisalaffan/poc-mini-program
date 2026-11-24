import SwiftUI
import UIKit

struct BenchmarkLabScreen: View {
    @Bindable var config: AppConfig
    let apiClient: APIClient
    let tokenManager: TokenManager
    let metrics: MetricsCollector
    let onDataChanged: () -> Void

    @State private var reconcileReport: ReconcileReport?
    @State private var statusMessage: String?
    @State private var isWorking = false

    var body: some View {
        ScrollView {
            VStack(spacing: 14) {
                GachaCard(title: "Server Tujuan") {
                    Picker("Server Tujuan", selection: $config.baseURLString) {
                        ForEach(BackendEndpointPreset.allCases) { preset in
                            Text(preset.label).tag(preset.rawValue)
                        }
                    }
                    .pickerStyle(.menu)
                    .onChange(of: config.baseURLString) { _, _ in
                        let newURL = config.baseURL
                        apiClient.updateBaseURL(newURL)
                        Task {
                            await tokenManager.updateBaseURL(newURL)
                            onDataChanged()
                        }
                        statusMessage = "Server diubah ke \(newURL.host ?? newURL.absoluteString)"
                    }

                    Text(config.baseURL.absoluteString)
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                }

                if let statusMessage {
                    Text(statusMessage)
                        .font(.footnote.weight(.medium))
                        .foregroundStyle(GachaTheme.primaryAccent)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(12)
                        .background(GachaTheme.primaryAccent.opacity(0.08), in: RoundedRectangle(cornerRadius: 12))
                }

                GachaCard(title: "Performa Klien") {
                    ForEach(ClientApproach.allCases) { approach in
                        let summary = metrics.summaries[approach] ?? ApproachMetricsSummary()
                        VStack(alignment: .leading, spacing: 8) {
                            Text(approach.displayTitle)
                                .font(.subheadline.weight(.semibold))
                            HStack(spacing: 8) {
                                MetricPill(title: "Avg TTI (ms)", value: formatMsNumber(summary.averageTTIMs))
                                MetricPill(title: "p95 TTI (ms)", value: formatMsNumber(summary.p95TTIMs))
                                MetricPill(title: "Avg FPS", value: summary.averageFPS.map { String(format: "%.0f", $0) } ?? "-")
                                MetricPill(title: "Jank", value: "\(summary.totalDroppedFrames)")
                            }
                            if approach == .webview, let rtt = summary.averageBridgeRttMs {
                                Text("Respon bridge rata-rata: \(String(format: "%.0f ms", rtt))")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                        .padding(.vertical, 4)

                        if approach != ClientApproach.allCases.last {
                            Divider()
                        }
                    }

                    HStack(spacing: 10) {
                        Button {
                            UIPasteboard.general.string = metrics.generateMarkdownReport()
                            statusMessage = "Tabel laporan disalin ke clipboard."
                        } label: {
                            Text("Salin Laporan")
                                .lineLimit(1)
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(GachaTheme.primaryAccent)

                        Button {
                            metrics.resetMetrics()
                            statusMessage = "Data performa lokal direset."
                        } label: {
                            Text("Reset")
                                .lineLimit(1)
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.bordered)
                    }
                    .padding(.top, 4)
                }

                GachaCard(title: "Simulasi & Rekonsiliasi") {
                    Toggle("Simulasi timeout transaksi berikutnya", isOn: $config.simulateTimeoutOnNextDraw)
                        .font(.subheadline)
                        .onChange(of: config.simulateTimeoutOnNextDraw) { _, newVal in
                            apiClient.setSimulateTimeoutOnNextDraw(newVal)
                        }

                    HStack(spacing: 10) {
                        Button {
                            Task { await runTopUp(500) }
                        } label: {
                            Text("Top-Up +500")
                                .lineLimit(1)
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.bordered)
                        .disabled(isWorking)

                        Button {
                            Task { await runReconcile() }
                        } label: {
                            Text("Cek Rekonsiliasi")
                                .lineLimit(1)
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(GachaTheme.primaryAccent)
                        .disabled(isWorking)
                    }

                    if let rep = reconcileReport {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(rep.balanced ? "Saldo seimbang" : "Saldo tidak seimbang")
                                .font(.subheadline.weight(.bold))
                                .foregroundStyle(rep.balanced ? GachaTheme.successColor : GachaTheme.dangerColor)
                            Text("Ledger \(rep.ledgerBalance) · Hitung ulang \(rep.computedBalance)")
                                .font(.caption.monospaced())
                            Text("Awal \(rep.totalSeeded) · Kuis +\(rep.totalQuizRewards) · Tukar -\(rep.totalDrawCost) · Hadiah +\(rep.totalRewardPoints)")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        .padding(.top, 4)
                    }
                }

                GachaCard(title: "Log Keamanan") {
                    if metrics.securityLogs.isEmpty {
                        Text("Belum ada aktivitas tercatat.")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(metrics.securityLogs.prefix(8)) { entry in
                            VStack(alignment: .leading, spacing: 3) {
                                HStack {
                                    Text(entry.source)
                                        .font(.caption.weight(.semibold))
                                    Spacer()
                                    Text(entry.decision)
                                        .font(.caption2.weight(.semibold))
                                        .foregroundStyle(
                                            entry.decision.contains("REJECT") || entry.decision.contains("BLOCK")
                                                ? GachaTheme.dangerColor
                                                : GachaTheme.successColor
                                        )
                                }
                                Text(entry.detail)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .padding(.vertical, 3)

                            if entry.id != metrics.securityLogs.prefix(8).last?.id {
                                Divider()
                            }
                        }
                    }
                }
            }
            .padding(16)
        }
        .background(GachaTheme.pageBackground)
        .navigationTitle("Lab & Benchmark")
        .navigationBarTitleDisplayMode(.inline)
        .onAppear {
            config.simulateTimeoutOnNextDraw = apiClient.isSimulateTimeoutArmed()
        }
    }

    private func formatMsNumber(_ value: Double?) -> String {
        guard let value else { return "-" }
        return String(format: "%.0f", value)
    }

    private func runTopUp(_ points: Int64) async {
        isWorking = true
        defer { isWorking = false }
        do {
            let bal = try await apiClient.syntheticTopUp(points: points)
            statusMessage = "Top-up berhasil. Saldo saat ini: \(bal.balance) poin."
            onDataChanged()
        } catch {
            statusMessage = "Top-up gagal: \(error.localizedDescription)"
        }
    }

    private func runReconcile() async {
        isWorking = true
        defer { isWorking = false }
        do {
            reconcileReport = try await apiClient.reconcile()
            statusMessage = "Rekonsiliasi selesai."
        } catch {
            statusMessage = "Rekonsiliasi gagal: \(error.localizedDescription)"
        }
    }
}

private struct MetricPill: View {
    let title: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
            Text(value)
                .font(.subheadline.weight(.bold).monospacedDigit())
                .lineLimit(1)
                .minimumScaleFactor(0.8)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
        .padding(.horizontal, 9)
        .padding(.vertical, 8)
        .background(GachaTheme.subtleBackground, in: RoundedRectangle(cornerRadius: 10))
    }
}
