import XCTest
@testable import GachaPOC

@MainActor
final class MetricsCollectorTests: XCTestCase {
    func testComputesAverageAndP95AndMarkdownReport() {
        let collector = MetricsCollector()
        for value in [100.0, 120.0, 150.0, 200.0, 400.0] {
            collector.recordTTI(approach: .native, durationMs: value)
        }
        collector.recordFPS(approach: .native, fps: 59.8, droppedFrames: 1)

        let summary = collector.summaries[.native]
        XCTAssertEqual(summary?.averageTTIMs, 194.0)
        XCTAssertEqual(summary?.p95TTIMs, 400.0)
        XCTAssertEqual(summary?.totalDroppedFrames, 1)

        let report = collector.generateMarkdownReport()
        XCTAssertTrue(report.contains("B · Native SwiftUI"))
        XCTAssertTrue(report.contains("194.0"))
    }
}
