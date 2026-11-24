import XCTest
@testable import GachaPOC

final class MockGachaAPI: GachaAPIServing, @unchecked Sendable {
    var balanceToReturn: Int64 = 1000
    var failNextDrawWith: GachaAPIError?
    var failNextQuizWith: GachaAPIError?
    var capturedIdempotencyKeys: [String] = []
    var capturedQuizKeys: [String] = []

    func getBalance(approach: ClientApproach) async throws -> BalanceResponse {
        BalanceResponse(userId: "demo-user-001", balance: balanceToReturn, lastLedgerId: 1, updatedAt: "")
    }

    func getProbabilities(approach: ClientApproach) async throws -> ProbabilitiesResponse {
        ProbabilitiesResponse(
            tableVersion: "v1.0.0",
            drawCost: 100,
            totalWeightBps: 10000,
            tiers: [
                RewardTier(id: "t1", code: "PTS_50", name: "50 Poin", rarity: .common, rewardPoints: 50, weightBps: 10000, probabilityPercent: 100, displayOrder: 1)
            ]
        )
    }

    func getHistory(limit: Int, approach: ClientApproach) async throws -> [HistoryItem] {
        []
    }

    func draw(idempotencyKey: String, expectedTableVersion: String?, approach: ClientApproach) async throws -> DrawResponse {
        capturedIdempotencyKeys.append(idempotencyKey)
        if let err = failNextDrawWith {
            failNextDrawWith = nil
            throw err
        }
        return DrawResponse(
            drawId: "draw-123",
            idempotencyKey: idempotencyKey,
            idempotencyReplayed: capturedIdempotencyKeys.count > 1,
            tableVersion: "v1.0.0",
            drawCost: 100,
            reward: RewardTier(id: "t1", code: "PTS_50", name: "50 Poin", rarity: .common, rewardPoints: 50, weightBps: 10000, probabilityPercent: 100, displayOrder: 1),
            balanceBefore: 1000,
            balanceAfter: 950,
            clientApproach: approach,
            createdAt: ""
        )
    }

    func syntheticTopUp(points: Int64) async throws -> BalanceResponse {
        BalanceResponse(userId: "demo-user-001", balance: balanceToReturn + points, lastLedgerId: 2, updatedAt: "")
    }

    func reconcile() async throws -> ReconcileReport {
        ReconcileReport(userId: "demo-user-001", ledgerBalance: 1000, computedBalance: 1000, totalSeeded: 1000, totalQuizRewards: 0, totalDrawCost: 0, totalRewardPoints: 0, totalDraws: 0, balanced: true)
    }

    func fetchSDUIScreen(variant: String, simulateSchemaVersion: Int?) async throws -> SDUIScreenPayload {
        SDUIScreenPayload(schemaVersion: 1, screenId: "gacha_home", variant: variant, navigationTitle: nil, backgroundAsset: nil, tableVersion: "v1.0.0", supportedActions: ["draw"], components: [], generatedAt: "")
    }

    func getVillageState(approach: ClientApproach) async throws -> VillageStateResponse {
        VillageStateResponse(
            userId: "demo-user-001",
            campaignBanner: "Mainkan kuis Livin' Village",
            balance: balanceToReturn,
            completedCount: 0,
            totalQuizzes: 1,
            quizzes: [
                VillageQuiz(
                    id: "quiz_tower",
                    buildingName: "Menara Utama",
                    title: "Kuis Keamanan",
                    pinColor: .blue,
                    xRatio: 0.63,
                    yRatio: 0.44,
                    rewardPoints: 150,
                    question: "Pertanyaan?",
                    options: ["A", "B"],
                    completed: false,
                    displayOrder: 1
                )
            ],
            rules: ["Aturan 1"]
        )
    }

    func submitQuiz(quizId: String, selectedIndex: Int, idempotencyKey: String, approach: ClientApproach) async throws -> QuizSubmitResponse {
        capturedQuizKeys.append(idempotencyKey)
        if let err = failNextQuizWith {
            failNextQuizWith = nil
            throw err
        }
        return QuizSubmitResponse(
            submissionId: "sub-1",
            quizId: quizId,
            idempotencyKey: idempotencyKey,
            idempotencyReplayed: capturedQuizKeys.count > 1,
            correct: true,
            rewardPoints: 150,
            balanceBefore: 1000,
            balanceAfter: 1150,
            explanation: "Benar",
            clientApproach: approach,
            createdAt: ""
        )
    }

    func fetchSDUIVillageScreen(variant: String, simulateSchemaVersion: Int?) async throws -> SDUIScreenPayload {
        SDUIScreenPayload(schemaVersion: 1, screenId: "livin_village_home", variant: variant, navigationTitle: "Livin' Village", backgroundAsset: "bg-isometric", tableVersion: "v1.0.0", supportedActions: ["open_quiz"], components: [], generatedAt: "")
    }
}

@MainActor
final class NativeGachaViewModelTests: XCTestCase {
    func testPreservesIdempotencyKeyAcrossNetworkTimeoutRetry() async {
        let mock = MockGachaAPI()
        mock.failNextDrawWith = .networkFailure("Simulated timeout")
        let metrics = MetricsCollector()
        let vm = NativeGachaViewModel(api: mock, metrics: metrics)

        await vm.loadAll()
        XCTAssertEqual(vm.balance, 1000)

        await vm.executeDraw()
        XCTAssertNotNil(vm.pendingIdempotencyKey)
        XCTAssertEqual(vm.errorBanner?.errorCode, "NETWORK_ERROR")

        await vm.executeDraw()
        XCTAssertEqual(mock.capturedIdempotencyKeys.count, 2)
        XCTAssertEqual(mock.capturedIdempotencyKeys[0], mock.capturedIdempotencyKeys[1])
        XCTAssertNil(vm.pendingIdempotencyKey)
        XCTAssertEqual(vm.balance, 950)
        XCTAssertEqual(vm.lastDraw?.idempotencyReplayed, true)
    }

    func testSubmitQuizPreservesIdempotencyKeyAcrossTimeoutRetry() async {
        let mock = MockGachaAPI()
        mock.failNextQuizWith = .networkFailure("Simulated quiz timeout")
        let metrics = MetricsCollector()
        let vm = NativeGachaViewModel(api: mock, metrics: metrics)

        await vm.loadAll()
        vm.selectedQuizOptionIndex = 1
        await vm.submitQuiz(quizId: "quiz_tower")
        XCTAssertNotNil(vm.pendingQuizIdempotencyKey)

        await vm.submitQuiz(quizId: "quiz_tower")
        XCTAssertEqual(mock.capturedQuizKeys.count, 2)
        XCTAssertEqual(mock.capturedQuizKeys[0], mock.capturedQuizKeys[1])
        XCTAssertNil(vm.pendingQuizIdempotencyKey)
        XCTAssertEqual(vm.balance, 1150)
    }

    func testClearsIdempotencyKeyOnInsufficientBalance() async {
        let mock = MockGachaAPI()
        mock.failNextDrawWith = .serverError(status: 422, code: "INSUFFICIENT_BALANCE", message: "Saldo kurang")
        let metrics = MetricsCollector()
        let vm = NativeGachaViewModel(api: mock, metrics: metrics)

        await vm.loadAll()
        await vm.executeDraw()

        XCTAssertNil(vm.pendingIdempotencyKey)
        XCTAssertEqual(vm.errorBanner?.errorCode, "INSUFFICIENT_BALANCE")
    }
}
