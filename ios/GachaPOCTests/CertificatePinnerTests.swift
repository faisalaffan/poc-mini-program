import XCTest
@testable import GachaPOC

final class CertificatePinnerTests: XCTestCase {
    func testAllowsTailscaleK3sAndLocalhostHosts() {
        let pinner = CertificatePinner(
            allowedHosts: [
                "gacha-poc.singapura-parore.ts.net",
                "vps-malang.singapura-parore.ts.net",
                "100.125.74.50",
                "macbook.singapura-parore.ts.net",
                "100.75.130.38",
                "127.0.0.1",
                "localhost"
            ],
            pinnedPublicKeyHashes: [],
            allowLocalDebugPlaintext: true
        )

        XCTAssertTrue(pinner.isHostAllowed("gacha-poc.singapura-parore.ts.net"))
        XCTAssertTrue(pinner.isHostAllowed("vps-malang.singapura-parore.ts.net"))
        XCTAssertTrue(pinner.isHostAllowed("macbook.singapura-parore.ts.net"))
        XCTAssertTrue(pinner.isHostAllowed("MACBOOK.SINGAPURA-PARORE.TS.NET"))
        XCTAssertTrue(pinner.isHostAllowed("127.0.0.1"))
        XCTAssertFalse(pinner.isHostAllowed("untrusted.example.com"))

        XCTAssertTrue(pinner.isDebugHost("gacha-poc.singapura-parore.ts.net"))
        XCTAssertTrue(pinner.isDebugHost("vps-malang.singapura-parore.ts.net"))
        XCTAssertTrue(pinner.isDebugHost("100.125.74.50"))
        XCTAssertTrue(pinner.isDebugHost("127.0.0.1"))
        XCTAssertTrue(pinner.isDebugHost("livin-village.faisalaffan.com"))
        XCTAssertFalse(pinner.isDebugHost("untrusted.example.com"))
    }
}
