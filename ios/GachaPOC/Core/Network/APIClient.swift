import Foundation

protocol GachaAPIServing: Sendable {
    func getBalance(approach: ClientApproach) async throws -> BalanceResponse
    func getProbabilities(approach: ClientApproach) async throws -> ProbabilitiesResponse
    func getHistory(limit: Int, approach: ClientApproach) async throws -> [HistoryItem]
    func draw(idempotencyKey: String, expectedTableVersion: String?, approach: ClientApproach) async throws -> DrawResponse
    func syntheticTopUp(points: Int64) async throws -> BalanceResponse
    func reconcile() async throws -> ReconcileReport
    func fetchSDUIScreen(variant: String, simulateSchemaVersion: Int?) async throws -> SDUIScreenPayload
    func getVillageState(approach: ClientApproach) async throws -> VillageStateResponse
    func submitQuiz(quizId: String, selectedIndex: Int, idempotencyKey: String, approach: ClientApproach) async throws -> QuizSubmitResponse
    func fetchSDUIVillageScreen(variant: String, simulateSchemaVersion: Int?) async throws -> SDUIScreenPayload
}

/// Shared network client untuk Pendekatan B & C menggunakan certificate pinning dan Keychain token management.
final class APIClient: GachaAPIServing, @unchecked Sendable {
    private let tokenManager: TokenManager
    private let session: URLSession
    private let lock = NSLock()
    private var baseURL: URL
    private var simulateTimeoutAfterServerCommitOnce = false

    init(baseURL: URL, tokenManager: TokenManager, pinner: CertificatePinner) {
        self.baseURL = baseURL
        self.tokenManager = tokenManager
        let config = URLSessionConfiguration.ephemeral
        config.timeoutIntervalForRequest = 10
        config.timeoutIntervalForResource = 15
        self.session = URLSession(configuration: config, delegate: pinner, delegateQueue: nil)
    }

    func updateBaseURL(_ newURL: URL) {
        lock.lock()
        defer { lock.unlock() }
        baseURL = newURL
    }

    private func currentBaseURL() -> URL {
        lock.lock()
        defer { lock.unlock() }
        return baseURL
    }

    func setSimulateTimeoutOnNextDraw(_ enabled: Bool) {
        lock.lock()
        defer { lock.unlock() }
        simulateTimeoutAfterServerCommitOnce = enabled
    }

    func isSimulateTimeoutArmed() -> Bool {
        lock.lock()
        defer { lock.unlock() }
        return simulateTimeoutAfterServerCommitOnce
    }

    private func consumeTimeoutSimulationFlag() -> Bool {
        lock.lock()
        defer { lock.unlock() }
        let current = simulateTimeoutAfterServerCommitOnce
        simulateTimeoutAfterServerCommitOnce = false
        return current
    }

    func getBalance(approach: ClientApproach) async throws -> BalanceResponse {
        try await request(path: "api/v1/gacha/balance", approach: approach)
    }

    func getProbabilities(approach: ClientApproach) async throws -> ProbabilitiesResponse {
        try await request(path: "api/v1/gacha/probabilities", approach: approach)
    }

    func getHistory(limit: Int = 20, approach: ClientApproach) async throws -> [HistoryItem] {
        let resp: HistoryResponse = try await request(
            path: "api/v1/gacha/history",
            queryItems: [URLQueryItem(name: "limit", value: "\(limit)")],
            approach: approach
        )
        return resp.items
    }

    func draw(
        idempotencyKey: String,
        expectedTableVersion: String?,
        approach: ClientApproach
    ) async throws -> DrawResponse {
        let shouldSimulateTimeout = consumeTimeoutSimulationFlag()
        let body = try JSONEncoder().encode(["expected_table_version": expectedTableVersion ?? ""])
        let response: DrawResponse = try await request(
            path: "api/v1/gacha/draw",
            method: "POST",
            body: body,
            extraHeaders: ["Idempotency-Key": idempotencyKey],
            approach: approach
        )

        // Simulasi timeout F6 setelah server memproses transaksi agar uji retry membuktikan idempotency replay.
        if shouldSimulateTimeout {
            throw GachaAPIError.networkFailure(
                "Simulasi timeout setelah server memproses draw. Ketuk Coba Lagi untuk menguji idempotency replay."
            )
        }
        return response
    }

    func syntheticTopUp(points: Int64) async throws -> BalanceResponse {
        let body = try JSONEncoder().encode(["points": points])
        return try await request(
            path: "api/v1/gacha/synthetic-topup",
            method: "POST",
            body: body,
            approach: .native
        )
    }

    func reconcile() async throws -> ReconcileReport {
        try await request(path: "api/v1/gacha/reconcile", approach: .native)
    }

    func fetchSDUIScreen(variant: String, simulateSchemaVersion: Int?) async throws -> SDUIScreenPayload {
        var items = [URLQueryItem(name: "variant", value: variant)]
        if let sim = simulateSchemaVersion {
            items.append(URLQueryItem(name: "simulate_schema_version", value: "\(sim)"))
        }
        return try await request(
            path: "api/v1/sdui/gacha-screen",
            queryItems: items,
            approach: .sdui
        )
    }

    func getVillageState(approach: ClientApproach) async throws -> VillageStateResponse {
        try await request(path: "api/v1/village/state", approach: approach)
    }

    func submitQuiz(
        quizId: String,
        selectedIndex: Int,
        idempotencyKey: String,
        approach: ClientApproach
    ) async throws -> QuizSubmitResponse {
        let shouldSimulateTimeout = consumeTimeoutSimulationFlag()
        let body = try JSONEncoder().encode(["selected_index": selectedIndex])
        let response: QuizSubmitResponse = try await request(
            path: "api/v1/village/quizzes/\(quizId)/submit",
            method: "POST",
            body: body,
            extraHeaders: ["Idempotency-Key": idempotencyKey],
            approach: approach
        )

        if shouldSimulateTimeout {
            throw GachaAPIError.networkFailure(
                "Simulasi timeout setelah server memproses kuis. Ketuk Coba Lagi untuk menguji idempotency replay."
            )
        }
        return response
    }

    func fetchSDUIVillageScreen(variant: String, simulateSchemaVersion: Int?) async throws -> SDUIScreenPayload {
        var items = [URLQueryItem(name: "variant", value: variant)]
        if let sim = simulateSchemaVersion {
            items.append(URLQueryItem(name: "simulate_schema_version", value: "\(sim)"))
        }
        return try await request(
            path: "api/v1/sdui/village-screen",
            queryItems: items,
            approach: .sdui
        )
    }

    private func request<T: Decodable>(
        path: String,
        method: String = "GET",
        queryItems: [URLQueryItem] = [],
        body: Data? = nil,
        extraHeaders: [String: String] = [:],
        approach: ClientApproach,
        allowSessionRefresh: Bool = true
    ) async throws -> T {
        var components = URLComponents(url: currentBaseURL().appending(path: path), resolvingAgainstBaseURL: false)
        if !queryItems.isEmpty {
            components?.queryItems = queryItems
        }
        guard let url = components?.url else {
            throw GachaAPIError.invalidURL
        }

        let token = try await tokenManager.ensureMasterSession()
        let deviceID = await tokenManager.getOrCreateDeviceID()

        var req = URLRequest(url: url, cachePolicy: .reloadIgnoringLocalCacheData, timeoutInterval: 10)
        req.httpMethod = method
        req.httpBody = body
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        req.setValue(deviceID, forHTTPHeaderField: "X-Device-ID")
        req.setValue(approach.rawValue, forHTTPHeaderField: "X-Client-Approach")
        if body != nil {
            req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        for (k, v) in extraHeaders {
            req.setValue(v, forHTTPHeaderField: k)
        }

        let data: Data
        let rawResp: URLResponse
        do {
            (data, rawResp) = try await session.data(for: req)
        } catch {
            throw GachaAPIError.networkFailure("Gangguan jaringan atau timeout: \(error.localizedDescription)")
        }

        guard let httpResp = rawResp as? HTTPURLResponse else {
            throw GachaAPIError.networkFailure("Respons HTTP tidak valid.")
        }

        if httpResp.statusCode == 401 && allowSessionRefresh {
            _ = try await tokenManager.refreshMasterSession()
            return try await request(
                path: path,
                method: method,
                queryItems: queryItems,
                body: body,
                extraHeaders: extraHeaders,
                approach: approach,
                allowSessionRefresh: false
            )
        }

        guard (200...299).contains(httpResp.statusCode) else {
            if let envelope = try? JSONDecoder().decode(APIErrorEnvelope.self, from: data) {
                throw GachaAPIError.serverError(
                    status: httpResp.statusCode,
                    code: envelope.error.code,
                    message: envelope.error.message
                )
            }
            throw GachaAPIError.serverError(
                status: httpResp.statusCode,
                code: "HTTP_\(httpResp.statusCode)",
                message: "Server mengembalikan status \(httpResp.statusCode)."
            )
        }

        do {
            return try JSONDecoder().decode(T.self, from: data)
        } catch {
            throw GachaAPIError.decodingFailure(error.localizedDescription)
        }
    }
}
