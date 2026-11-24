import XCTest
@testable import GachaPOC

@MainActor
final class NativeBridgeSecurityTests: XCTestCase {
    private let allowedHosts: Set<String> = ["127.0.0.1", "gacha-poc.apps.ocp.internal"]

    func testWhitelistedGetSessionTokenFromMainFrameSucceeds() {
        let body: [String: Any] = [
            "requestId": "req-1",
            "method": "getSessionToken"
        ]
        let result = NativeBridgeHandler.validateMessage(
            body: body,
            originHost: "127.0.0.1",
            isMainFrame: true,
            allowedHosts: allowedHosts
        )

        guard case let .success(call) = result else {
            return XCTFail("Expected valid bridge call to succeed")
        }
        XCTAssertEqual(call.method, .getSessionToken)
        XCTAssertEqual(call.requestId, "req-1")
    }

    func testNonWhitelistedMethodIsRejected() {
        let body: [String: Any] = [
            "requestId": "req-2",
            "method": "stealAllKeychainSecrets"
        ]
        let result = NativeBridgeHandler.validateMessage(
            body: body,
            originHost: "127.0.0.1",
            isMainFrame: true,
            allowedHosts: allowedHosts
        )

        XCTAssertEqual(result, .failure(.methodNotAllowed("stealAllKeychainSecrets")))
    }

    func testUntrustedOriginIsRejected() {
        let body: [String: Any] = [
            "requestId": "req-3",
            "method": "getSessionToken"
        ]
        let result = NativeBridgeHandler.validateMessage(
            body: body,
            originHost: "evil-phishing.example.com",
            isMainFrame: true,
            allowedHosts: allowedHosts
        )

        XCTAssertEqual(result, .failure(.untrustedOrigin("evil-phishing.example.com")))
    }

    func testIframeCallIsRejected() {
        let body: [String: Any] = [
            "requestId": "req-4",
            "method": "getSessionToken"
        ]
        let result = NativeBridgeHandler.validateMessage(
            body: body,
            originHost: "127.0.0.1",
            isMainFrame: false,
            allowedHosts: allowedHosts
        )

        XCTAssertEqual(result, .failure(.nonMainFrame))
    }
}
