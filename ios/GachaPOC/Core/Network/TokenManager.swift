import Foundation

actor TokenManager {
    private enum Keys {
        static let masterToken = "gacha.master_jwt"
        static let deviceID = "gacha.device_id"
        static let userID = "gacha.user_id"
    }

    private let keychain: KeychainStoring
    private var baseURL: URL
    private let session: URLSession

    init(keychain: KeychainStoring, baseURL: URL, pinner: CertificatePinner) {
        self.keychain = keychain
        self.baseURL = baseURL
        let config = URLSessionConfiguration.ephemeral
        config.timeoutIntervalForRequest = 10
        config.timeoutIntervalForResource = 15
        self.session = URLSession(configuration: config, delegate: pinner, delegateQueue: nil)
    }

    func updateBaseURL(_ newURL: URL) {
        if self.baseURL != newURL {
            self.baseURL = newURL
            keychain.delete(forKey: Keys.masterToken)
        }
    }

    func getOrCreateDeviceID() -> String {
        if let existing = keychain.loadString(forKey: Keys.deviceID), !existing.isEmpty {
            return existing
        }
        let created = "ios-poc-\(UUID().uuidString.lowercased())"
        try? keychain.saveString(created, forKey: Keys.deviceID)
        return created
    }

    func currentMasterToken() -> String? {
        keychain.loadString(forKey: Keys.masterToken)
    }

    func ensureMasterSession(userID: String = "demo-user-001") async throws -> String {
        if let existing = keychain.loadString(forKey: Keys.masterToken),
           keychain.loadString(forKey: Keys.userID) == userID {
            return existing
        }
        return try await refreshMasterSession(userID: userID)
    }

    @discardableResult
    func refreshMasterSession(userID: String = "demo-user-001") async throws -> String {
        let deviceID = getOrCreateDeviceID()
        let url = baseURL.appending(path: "api/v1/auth/session")
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.setValue(deviceID, forHTTPHeaderField: "X-Device-ID")
        req.httpBody = try JSONEncoder().encode(["user_id": userID, "device_id": deviceID])

        let (data, response) = try await session.data(for: req)
        guard let httpResp = response as? HTTPURLResponse, (200...299).contains(httpResp.statusCode) else {
            throw GachaAPIError.missingSession
        }
        let decoded = try JSONDecoder().decode(SessionResponse.self, from: data)
        try keychain.saveString(decoded.token, forKey: Keys.masterToken)
        try keychain.saveString(decoded.userId, forKey: Keys.userID)
        return decoded.token
    }

    /// Menukar master token di Keychain menjadi short-lived token (TTL 120s) untuk diserahkan lewat Native Bridge (F7).
    func mintShortLivedWebViewToken() async throws -> WebViewTokenResponse {
        let master = try await ensureMasterSession()
        let deviceID = getOrCreateDeviceID()
        let url = baseURL.appending(path: "api/v1/auth/webview-token")
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("Bearer \(master)", forHTTPHeaderField: "Authorization")
        req.setValue(deviceID, forHTTPHeaderField: "X-Device-ID")

        let (data, response) = try await session.data(for: req)
        if let httpResp = response as? HTTPURLResponse, httpResp.statusCode == 401 {
            let refreshed = try await refreshMasterSession()
            req.setValue("Bearer \(refreshed)", forHTTPHeaderField: "Authorization")
            let (retryData, retryResp) = try await session.data(for: req)
            guard let retryHttp = retryResp as? HTTPURLResponse, (200...299).contains(retryHttp.statusCode) else {
                throw GachaAPIError.missingSession
            }
            return try JSONDecoder().decode(WebViewTokenResponse.self, from: retryData)
        }
        guard let httpResp = response as? HTTPURLResponse, (200...299).contains(httpResp.statusCode) else {
            throw GachaAPIError.missingSession
        }
        return try JSONDecoder().decode(WebViewTokenResponse.self, from: data)
    }

    func clearMasterSession() {
        keychain.delete(forKey: Keys.masterToken)
    }
}
