import Foundation
import Observation

enum NativeGachaUIState: Equatable, Sendable {
    case loading
    case ready
    case drawing
    case animating(DrawResponse)
    case showingResult(DrawResponse)
}

enum VillageSheetRoute: Identifiable, Equatable, Sendable {
    case quiz(String)
    case help
    case rewards

    var id: String {
        switch self {
        case let .quiz(quizId): return "quiz:\(quizId)"
        case .help: return "help"
        case .rewards: return "rewards"
        }
    }
}

@MainActor
@Observable
final class NativeGachaViewModel {
    private let api: GachaAPIServing
    private let metrics: MetricsCollector

    private(set) var uiState: NativeGachaUIState = .loading
    private(set) var balance: Int64 = 0
    private(set) var villageState: VillageStateResponse?
    private(set) var probabilities: ProbabilitiesResponse?
    private(set) var history: [HistoryItem] = []
    private(set) var lastDraw: DrawResponse?
    private(set) var lastQuizResult: QuizSubmitResponse?
    private(set) var pendingIdempotencyKey: String?
    private(set) var pendingQuizIdempotencyKey: String?
    private(set) var errorBanner: GachaAPIError?
    private(set) var isSubmittingQuiz = false

    var activeSheet: VillageSheetRoute?
    var selectedQuizOptionIndex: Int?

    init(api: GachaAPIServing, metrics: MetricsCollector) {
        self.api = api
        self.metrics = metrics
    }

    var isBusy: Bool {
        switch uiState {
        case .loading, .drawing, .animating:
            return true
        case .ready, .showingResult:
            return false
        }
    }

    var canDraw: Bool {
        guard !isBusy, let probabilities else { return false }
        return balance >= probabilities.drawCost || pendingIdempotencyKey != nil
    }

    func quiz(for id: String) -> VillageQuiz? {
        villageState?.quizzes.first(where: { $0.id == id })
    }

    func openQuiz(_ quiz: VillageQuiz) {
        selectedQuizOptionIndex = nil
        lastQuizResult = nil
        pendingQuizIdempotencyKey = nil
        errorBanner = nil
        activeSheet = .quiz(quiz.id)
    }

    func openHelp() {
        errorBanner = nil
        activeSheet = .help
    }

    func openRewards() {
        errorBanner = nil
        activeSheet = .rewards
    }

    func closeSheet() {
        activeSheet = nil
    }

    func loadAll() async {
        let start = CFAbsoluteTimeGetCurrent()
        uiState = .loading
        errorBanner = nil
        do {
            async let villageReq = api.getVillageState(approach: .native)
            async let probReq = api.getProbabilities(approach: .native)
            async let histReq = api.getHistory(limit: 15, approach: .native)

            let (state, prob, hist) = try await (villageReq, probReq, histReq)
            villageState = state
            balance = state.balance
            probabilities = prob
            history = hist
            uiState = .ready

            let elapsedMs = (CFAbsoluteTimeGetCurrent() - start) * 1000.0
            metrics.recordTTI(approach: .native, durationMs: (elapsedMs * 10).rounded() / 10)
        } catch let err as GachaAPIError {
            errorBanner = err
            uiState = .ready
        } catch {
            errorBanner = .networkFailure(error.localizedDescription)
            uiState = .ready
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
                approach: .native
            )
            pendingQuizIdempotencyKey = nil
            lastQuizResult = res
            balance = res.balanceAfter

            let latencyMs = (CFAbsoluteTimeGetCurrent() - reqStart) * 1000.0
            metrics.recordDrawLatency(approach: .native, durationMs: (latencyMs * 10).rounded() / 10)

            if res.correct {
                if var state = villageState {
                    state.balance = res.balanceAfter
                    for idx in state.quizzes.indices where state.quizzes[idx].id == res.quizId {
                        state.quizzes[idx].completed = true
                    }
                    state.completedCount = state.quizzes.filter(\.completed).count
                    villageState = state
                }
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
        guard !isBusy else { return }
        errorBanner = nil
        uiState = .drawing

        let key = pendingIdempotencyKey ?? UUID().uuidString.lowercased()
        pendingIdempotencyKey = key
        let drawStart = CFAbsoluteTimeGetCurrent()

        do {
            let result = try await api.draw(
                idempotencyKey: key,
                expectedTableVersion: probabilities?.tableVersion,
                approach: .native
            )
            pendingIdempotencyKey = nil
            lastDraw = result
            balance = result.balanceAfter
            if var state = villageState {
                state.balance = result.balanceAfter
                villageState = state
            }

            let drawMs = (CFAbsoluteTimeGetCurrent() - drawStart) * 1000.0
            metrics.recordDrawLatency(approach: .native, durationMs: (drawMs * 10).rounded() / 10)

            uiState = .animating(result)
            if let updatedHistory = try? await api.getHistory(limit: 15, approach: .native) {
                history = updatedHistory
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
            uiState = .ready
        } catch {
            errorBanner = .networkFailure(error.localizedDescription)
            uiState = .ready
        }
    }

    func completeAnimation(for result: DrawResponse) {
        uiState = .showingResult(result)
    }
}
