import CryptoKit
import Foundation
import Security

/// Memvalidasi certificate pinning berbasis SHA-256 public key untuk domain backend OCP & WebView allowlist.
final class CertificatePinner: NSObject, URLSessionDelegate, Sendable {
    let allowedHosts: Set<String>
    let pinnedPublicKeyHashes: Set<String>
    let allowLocalDebugPlaintext: Bool

    init(
        allowedHosts: Set<String>,
        pinnedPublicKeyHashes: Set<String>,
        allowLocalDebugPlaintext: Bool
    ) {
        self.allowedHosts = allowedHosts
        self.pinnedPublicKeyHashes = pinnedPublicKeyHashes
        self.allowLocalDebugPlaintext = allowLocalDebugPlaintext
    }

    func isHostAllowed(_ host: String?) -> Bool {
        guard let host else { return false }
        return allowedHosts.contains(host.lowercased())
    }

    func evaluate(challenge: URLAuthenticationChallenge) -> (URLSession.AuthChallengeDisposition, URLCredential?) {
        let host = challenge.protectionSpace.host.lowercased()
        let authMethod = challenge.protectionSpace.authenticationMethod

        guard isHostAllowed(host) else {
            return (.cancelAuthenticationChallenge, nil)
        }

        if authMethod != NSURLAuthenticationMethodServerTrust {
            return (.performDefaultHandling, nil)
        }

        guard let trust = challenge.protectionSpace.serverTrust else {
            return (.cancelAuthenticationChallenge, nil)
        }

        var trustError: CFError?
        guard SecTrustEvaluateWithError(trust, &trustError) else {
            return (.cancelAuthenticationChallenge, nil)
        }

        // Jika pin public key SHA-256 dikonfigurasi, wajib cocok dengan chain sertifikat.
        // Jika kosong (mis. Cloudflare Universal SSL dengan rotasi otomatis), koneksi tetap
        // divalidasi oleh host allowlist + SecTrustEvaluateWithError dari sistem iOS.
        if !pinnedPublicKeyHashes.isEmpty {
            let hashes = extractPublicKeyHashes(from: trust)
            if hashes.isDisjoint(with: pinnedPublicKeyHashes) {
                return (.cancelAuthenticationChallenge, nil)
            }
        }

        return (.useCredential, URLCredential(trust: trust))
    }

    func urlSession(
        _ session: URLSession,
        didReceive challenge: URLAuthenticationChallenge
    ) async -> (URLSession.AuthChallengeDisposition, URLCredential?) {
        evaluate(challenge: challenge)
    }

    func isDebugHost(_ host: String) -> Bool {
        let lower = host.lowercased()
        return lower == "127.0.0.1"
            || lower == "localhost"
            || lower == "100.75.130.38"
            || lower == "100.125.74.50"
            || lower == "livin-village.faisalaffan.com"
            || lower.hasSuffix(".ts.net")
    }

    private func extractPublicKeyHashes(from trust: SecTrust) -> Set<String> {
        guard let chain = SecTrustCopyCertificateChain(trust) as? [SecCertificate] else {
            return []
        }
        var result: Set<String> = []
        for cert in chain {
            guard let key = SecCertificateCopyKey(cert),
                  let keyData = SecKeyCopyExternalRepresentation(key, nil) as Data? else {
                continue
            }
            let digest = SHA256.hash(data: keyData)
            let b64 = Data(digest).base64EncodedString()
            result.insert(b64)
        }
        return result
    }
}
