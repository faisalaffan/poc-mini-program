import Foundation
import Observation
import QuartzCore

struct ApproachMetricsSummary: Equatable, Sendable {
    var ttiSamplesMs: [Double] = []
    var bridgeRttSamplesMs: [Double] = []
    var drawLatencySamplesMs: [Double] = []
    var fpsSamples: [Double] = []
    var totalDroppedFrames: Int = 0
    var drawCount: Int = 0
    var lastUpdated: Date?

    var averageTTIMs: Double? {
        Self.average(ttiSamplesMs)
    }

    var p95TTIMs: Double? {
        Self.percentile95(ttiSamplesMs)
    }

    var averageBridgeRttMs: Double? {
        Self.average(bridgeRttSamplesMs)
    }

    var averageFPS: Double? {
        Self.average(fpsSamples)
    }

    static func average(_ values: [Double]) -> Double? {
        guard !values.isEmpty else { return nil }
        return values.reduce(0, +) / Double(values.count)
    }

    static func percentile95(_ values: [Double]) -> Double? {
        guard !values.isEmpty else { return nil }
        let sorted = values.sorted()
        let index = min(sorted.count - 1, max(0, Int(ceil(Double(sorted.count) * 0.95)) - 1))
        return sorted[index]
    }
}

struct SecurityAuditEntry: Identifiable, Sendable {
    let id = UUID()
    let timestamp: Date
    let source: String
    let decision: String
    let detail: String
}

@MainActor
@Observable
final class MetricsCollector {
    private(set) var summaries: [ClientApproach: ApproachMetricsSummary] = [
        .webview: ApproachMetricsSummary(),
        .native: ApproachMetricsSummary(),
        .sdui: ApproachMetricsSummary()
    ]
    private(set) var securityLogs: [SecurityAuditEntry] = []

    private var displayLink: CADisplayLink?
    private var frameTimestamps: [CFTimeInterval] = []
    private var activeFPSApproach: ClientApproach?

    func recordTTI(approach: ClientApproach, durationMs: Double) {
        guard durationMs >= 0 else { return }
        var summary = summaries[approach] ?? ApproachMetricsSummary()
        summary.ttiSamplesMs.append(durationMs)
        summary.lastUpdated = Date()
        summaries[approach] = summary
    }

    func recordBridgeRTT(durationMs: Double) {
        guard durationMs >= 0 else { return }
        var summary = summaries[.webview] ?? ApproachMetricsSummary()
        summary.bridgeRttSamplesMs.append(durationMs)
        summary.lastUpdated = Date()
        summaries[.webview] = summary
    }

    func recordDrawLatency(approach: ClientApproach, durationMs: Double) {
        guard durationMs >= 0 else { return }
        var summary = summaries[approach] ?? ApproachMetricsSummary()
        summary.drawLatencySamplesMs.append(durationMs)
        summary.drawCount += 1
        summary.lastUpdated = Date()
        summaries[approach] = summary
    }

    func recordFPS(approach: ClientApproach, fps: Double, droppedFrames: Int) {
        guard fps > 0 else { return }
        var summary = summaries[approach] ?? ApproachMetricsSummary()
        summary.fpsSamples.append(fps)
        summary.totalDroppedFrames += max(0, droppedFrames)
        summary.lastUpdated = Date()
        summaries[approach] = summary
    }

    func startDisplayLinkFPS(for approach: ClientApproach) {
        stopDisplayLink()
        frameTimestamps = []
        activeFPSApproach = approach
        let link = CADisplayLink(target: self, selector: #selector(onDisplayLinkTick(_:)))
        link.add(to: .main, forMode: .common)
        displayLink = link
    }

    func finishDisplayLinkFPS() {
        guard let approach = activeFPSApproach else {
            stopDisplayLink()
            return
        }
        let timestamps = frameTimestamps
        stopDisplayLink()

        guard timestamps.count >= 2,
              let first = timestamps.first,
              let last = timestamps.last,
              last > first else {
            return
        }
        let totalDuration = last - first
        let avgFPS = Double(timestamps.count - 1) / totalDuration
        var dropped = 0
        for idx in 1..<timestamps.count {
            let delta = timestamps[idx] - timestamps[idx - 1]
            if delta > 0.020 {
                dropped += 1
            }
        }
        recordFPS(approach: approach, fps: (avgFPS * 10).rounded() / 10, droppedFrames: dropped)
    }

    private func stopDisplayLink() {
        displayLink?.invalidate()
        displayLink = nil
        activeFPSApproach = nil
        frameTimestamps = []
    }

    @objc private func onDisplayLinkTick(_ link: CADisplayLink) {
        frameTimestamps.append(link.timestamp)
    }

    func recordSecurityEvent(source: String, decision: String, detail: String) {
        securityLogs.insert(
            SecurityAuditEntry(timestamp: Date(), source: source, decision: decision, detail: detail),
            at: 0
        )
        if securityLogs.count > 50 {
            securityLogs.removeLast()
        }
    }

    func resetMetrics() {
        for approach in ClientApproach.allCases {
            summaries[approach] = ApproachMetricsSummary()
        }
        securityLogs.removeAll()
    }

    func generateMarkdownReport() -> String {
        var lines = [
            "| Pendekatan | Avg TTI (ms) | p95 TTI (ms) | Avg FPS | Jank Frames | Draw Count |",
            "| --- | --- | --- | --- | --- | --- |"
        ]
        for approach in ClientApproach.allCases {
            let s = summaries[approach] ?? ApproachMetricsSummary()
            let avgTTI = s.averageTTIMs.map { String(format: "%.1f", $0) } ?? "-"
            let p95TTI = s.p95TTIMs.map { String(format: "%.1f", $0) } ?? "-"
            let fps = s.averageFPS.map { String(format: "%.1f", $0) } ?? "-"
            lines.append("| \(approach.displayTitle) | \(avgTTI) | \(p95TTI) | \(fps) | \(s.totalDroppedFrames) | \(s.drawCount) |")
        }
        return lines.joined(separator: "\n")
    }
}
