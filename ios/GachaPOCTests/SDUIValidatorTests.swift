import XCTest
@testable import GachaPOC

final class SDUIValidatorTests: XCTestCase {
    func testValidVillageSchemaV1Passes() {
        let payload = SDUIScreenPayload(
            schemaVersion: 1,
            screenId: "livin_village_home",
            variant: "default",
            navigationTitle: "Livin' Village",
            backgroundAsset: "bg-isometric",
            tableVersion: "v1.0.0",
            supportedActions: ["open_quiz", "open_help", "open_rewards"],
            components: [
                SDUINodePayload(id: "banner", type: "floating_banner", props: ["text": .string("Mainkan kuis")], action: nil),
                SDUINodePayload(id: "fab_help", type: "floating_action", props: ["icon": .string("help")], action: "open_help"),
                SDUINodePayload(id: "pin_tower", type: "map_pin", props: ["x_ratio": .double(0.63), "y_ratio": .double(0.44), "quiz_id": .string("quiz_tower")], action: "open_quiz")
            ],
            generatedAt: ""
        )

        let res = SDUIValidator.validate(payload: payload)
        guard case let .success(screen) = res else {
            return XCTFail("Expected village schema v1 to pass validation")
        }
        XCTAssertEqual(screen.backgroundAsset, "bg-isometric")
        XCTAssertEqual(screen.nodes.count, 3)
        XCTAssertEqual(screen.nodes[2].type, .mapPin)
        XCTAssertEqual(screen.nodes[2].action, .openQuiz)
    }

    func testUnsupportedFutureSchemaTriggersFallbackError() {
        let payload = SDUIScreenPayload(
            schemaVersion: 2,
            screenId: "livin_village_home",
            variant: "default",
            navigationTitle: "Livin' Village",
            backgroundAsset: "bg-isometric",
            tableVersion: "v1.0.0",
            supportedActions: ["open_quiz"],
            components: [
                SDUINodePayload(id: "banner", type: "floating_banner", props: nil, action: nil)
            ],
            generatedAt: ""
        )

        let res = SDUIValidator.validate(payload: payload)
        XCTAssertEqual(res, .failure(.unsupportedSchemaVersion(received: 2, maxSupported: 1)))
    }

    func testUnknownComponentTypeIsRejected() {
        let payload = SDUIScreenPayload(
            schemaVersion: 1,
            screenId: "livin_village_home",
            variant: "default",
            navigationTitle: "Livin' Village",
            backgroundAsset: "bg-isometric",
            tableVersion: "v1.0.0",
            supportedActions: ["open_quiz"],
            components: [
                SDUINodePayload(id: "x1", type: "arbitrary_webview_inject", props: nil, action: nil)
            ],
            generatedAt: ""
        )

        let res = SDUIValidator.validate(payload: payload)
        XCTAssertEqual(res, .failure(.unknownComponent(nodeId: "x1", typeName: "arbitrary_webview_inject")))
    }

    func testNonWhitelistedActionIsRejected() {
        let payload = SDUIScreenPayload(
            schemaVersion: 1,
            screenId: "livin_village_home",
            variant: "default",
            navigationTitle: "Livin' Village",
            backgroundAsset: "bg-isometric",
            tableVersion: "v1.0.0",
            supportedActions: ["open_quiz"],
            components: [
                SDUINodePayload(id: "btn1", type: "floating_action", props: nil, action: "open_external_url")
            ],
            generatedAt: ""
        )

        let res = SDUIValidator.validate(payload: payload)
        XCTAssertEqual(res, .failure(.disallowedAction(nodeId: "btn1", actionName: "open_external_url")))
    }
}
