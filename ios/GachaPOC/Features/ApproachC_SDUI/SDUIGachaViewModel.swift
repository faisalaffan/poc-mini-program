import Foundation
import Observation

@MainActor
@Observable
final class SDUIGachaViewModel {
    private let api: GachaAPIServing
    private let metrics: MetricsCollector
    private let config: AppConfig

    private(set) var loading = false
    private(set) var drawing = false
    private(set) var animating = false
    private(set) var isSubmittingQuiz = false
    private(set) var validatedScreen: ValidatedSDUIScreen?
    private(set) var fallbackReason: SDUIValidationError?
    private(set) var villageState: VillageStateResponse?
    private(set) var balance: Int64 = 0
    private(set) var probabilities: ProbabilitiesResponse?
    private(set) var history: [HistoryItem] = []
    private(set) var lastDraw: DrawResponse?
    private(set) var lastQuizResult: QuizSubmitResponse?
    private(set) var pendingIdempotencyKey: String?
    private(set) var pendingQuizIdempotencyKey: String?
    private(set) var errorBanner: GachaAPIError?

    var activeSheet: VillageSheetRoute?
    var selectedQuizOptionIndex: Int?
    private(set) var activeQuizNodeID: String?

    private var supportingDataTask: Task<Void, Never>?

    init(api: GachaAPIServing, metrics: MetricsCollector, config: AppConfig) {
        self.api = api
        self.metrics = metrics
        self.config = config
    }

    func quiz(for id: String, nodeID: String? = nil) -> VillageQuiz? {
        let targetNodeID = nodeID ?? activeQuizNodeID
        let matchedNode = validatedScreen?.nodes.first(where: {
            $0.type == .mapPin && (targetNodeID != nil ? $0.id == targetNodeID : $0.props["quiz_id"]?.stringValue == id)
        }) ?? validatedScreen?.nodes.first(where: {
            $0.type == .mapPin && $0.props["quiz_id"]?.stringValue == id
        })

        if let node = matchedNode,
           let question = node.props["question"]?.stringValue,
           !question.isEmpty,
           let options = node.props["options"]?.arrayValue?.compactMap(\.stringValue),
           !options.isEmpty
        {
            return VillageQuiz(
                id: id,
                buildingName: node.props["building_name"]?.stringValue ?? node.props["title"]?.stringValue ?? "Kuis",
                title: node.props["title"]?.stringValue ?? "Kuis",
                pinColor: PinColor(rawValue: node.props["pin_color"]?.stringValue ?? "blue") ?? .blue,
                xRatio: node.props["x_ratio"]?.doubleValue ?? 0.5,
                yRatio: node.props["y_ratio"]?.doubleValue ?? 0.5,
                rewardPoints: Int64(node.props["reward_points"]?.intValue ?? 150),
                question: question,
                options: options,
                completed: node.props["completed"]?.boolValue ?? false,
                displayOrder: 0
            )
        }
        return villageState?.quizzes.first(where: { $0.id == id })
    }

    func loadAll(resetScreen: Bool = false) async {
        let start = CFAbsoluteTimeGetCurrent()
        loading = true
        errorBanner = nil
        fallbackReason = nil
        if resetScreen {
            validatedScreen = nil
        }

        let simulatedVersion = config.simulateUnsupportedSDUISchema ? 2 : nil

        // Jalankan fetch data pendukung (state, probabilities, history) di background agar tidak memblokir render awal SDUI.
        supportingDataTask?.cancel()
        supportingDataTask = Task { [weak self] in
            guard let self else { return }
            async let stateReq = self.api.getVillageState(approach: .sdui)
            async let probReq = self.api.getProbabilities(approach: .sdui)
            async let histReq = self.api.getHistory(limit: 15, approach: .sdui)

            if let state = try? await stateReq {
                guard !Task.isCancelled else { return }
                self.villageState = state
                self.balance = state.balance
            }
            if let prob = try? await probReq {
                guard !Task.isCancelled else { return }
                self.probabilities = prob
            }
            if let hist = try? await histReq {
                guard !Task.isCancelled else { return }
                self.history = hist
            }
        }

        do {
            let rawScreen = try await api.fetchSDUIVillageScreen(
                variant: config.sduiVariant,
                simulateSchemaVersion: simulatedVersion
            )

            switch SDUIValidator.validate(payload: rawScreen) {
            case let .success(screen):
                validatedScreen = screen
                fallbackReason = nil
                metrics.recordSecurityEvent(
                    source: "ApproachC.SDUI",
                    decision: "VALIDATED",
                    detail: "Village schema v\(screen.schemaVersion) '\(screen.variant)' (\(screen.nodes.count) komponen dirangkai)"
                )
            case let .failure(err):
                validatedScreen = nil
                fallbackReason = err
                metrics.recordSecurityEvent(
                    source: "ApproachC.SDUI",
                    decision: "FALLBACK_TRIGGERED",
                    detail: err.reasonText
                )
            }

            let elapsedMs = (CFAbsoluteTimeGetCurrent() - start) * 1000.0
            metrics.recordTTI(approach: .sdui, durationMs: (elapsedMs * 10).rounded() / 10)
        } catch let err as GachaAPIError {
            errorBanner = err
        } catch {
            errorBanner = .networkFailure(error.localizedDescription)
        }

        loading = false
    }

    func dispatch(node: ValidatedSDUINode) async {
        guard let action = node.action else { return }
        switch action {
        case .openQuiz:
            let quizId = node.props["quiz_id"]?.stringValue ?? ""
            guard !quizId.isEmpty else { return }
            activeQuizNodeID = node.id
            if quiz(for: quizId, nodeID: node.id) == nil {
                await supportingDataTask?.value
            }
            selectedQuizOptionIndex = nil
            lastQuizResult = nil
            pendingQuizIdempotencyKey = nil
            errorBanner = nil
            activeSheet = .quiz(quizId)
        case .openHelp:
            if villageState == nil || probabilities == nil {
                await supportingDataTask?.value
            }
            errorBanner = nil
            activeSheet = .help
        case .openRewards:
            if villageState == nil {
                await supportingDataTask?.value
            }
            errorBanner = nil
            activeSheet = .rewards
        case .draw:
            await executeDraw()
        case .refresh:
            await loadAll()
        case .toggleHistory:
            break
        }
    }

    func submitQuiz(quizId: String) async {
        guard !isSubmittingQuiz, let selectedIdx = selectedQuizOptionIndex else { return }
        isSubmittingQuiz = true
        errorBanner = nil

        let key = pendingQuizIdempotencyKey ?? UUID().uuidString.lowercased()
        pendingQuizIdempotencyKey = key
        let reqStart = CFAbsoluteTimeGetCurrent()

        do {
            let res = try await api.submitQuiz(
                quizId: quizId,
                selectedIndex: selectedIdx,
                idempotencyKey: key,
                approach: .sdui
            )
            pendingQuizIdempotencyKey = nil
            lastQuizResult = res
            balance = res.balanceAfter

            let latencyMs = (CFAbsoluteTimeGetCurrent() - reqStart) * 1000.0
            metrics.recordDrawLatency(approach: .sdui, durationMs: (latencyMs * 10).rounded() / 10)

            if res.correct {
                await loadAll()
            }
        } catch let err as GachaAPIError {
            if err.errorCode == "QUIZ_ALREADY_COMPLETED" || err.errorCode == "IDEMPOTENCY_PAYLOAD_MISMATCH" {
                pendingQuizIdempotencyKey = nil
            }
            errorBanner = err
        } catch {
            errorBanner = .networkFailure(error.localizedDescription)
        }

        isSubmittingQuiz = false
    }

    func executeDraw() async {
        guard !drawing, !animating else { return }
        drawing = true
        errorBanner = nil

        let key = pendingIdempotencyKey ?? UUID().uuidString.lowercased()
        pendingIdempotencyKey = key
        let drawStart = CFAbsoluteTimeGetCurrent()

        do {
            let result = try await api.draw(
                idempotencyKey: key,
                expectedTableVersion: probabilities?.tableVersion,
                approach: .sdui
            )
            pendingIdempotencyKey = nil
            lastDraw = result
            balance = result.balanceAfter
            if var state = villageState {
                state.balance = result.balanceAfter
                villageState = state
            }
            animating = true

            let drawMs = (CFAbsoluteTimeGetCurrent() - drawStart) * 1000.0
            metrics.recordDrawLatency(approach: .sdui, durationMs: (drawMs * 10).rounded() / 10)

            if let updated = try? await api.getHistory(limit: 15, approach: .sdui) {
                history = updated
            }
        } catch let err as GachaAPIError {
            switch err.errorCode {
            case "INSUFFICIENT_BALANCE", "IDEMPOTENCY_PAYLOAD_MISMATCH":
                pendingIdempotencyKey = nil
            case "TABLE_VERSION_MISMATCH":
                pendingIdempotencyKey = nil
                await loadAll()
            default:
                break
            }
            errorBanner = err
        } catch {
            errorBanner = .networkFailure(error.localizedDescription)
        }

        drawing = false
    }

    func completeAnimation() {
        animating = false
    }

    func closeSheet() {
        activeSheet = nil
        activeQuizNodeID = nil
    }
}
