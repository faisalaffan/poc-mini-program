import Foundation
import Security

protocol KeychainStoring: Sendable {
    func saveString(_ value: String, forKey key: String) throws
    func loadString(forKey key: String) -> String?
    func delete(forKey key: String)
}

enum KeychainError: Error, Sendable {
    case unexpectedStatus(OSStatus)
    case encodingFailed
}

/// Wrapper aman Security.framework agar token & device_id tidak pernah disimpan di UserDefaults atau Web Storage.
final class KeychainStore: KeychainStoring, @unchecked Sendable {
    private let service: String
    private let lock = NSLock()

    init(service: String = "id.co.poc.gacha.keychain") {
        self.service = service
    }

    func saveString(_ value: String, forKey key: String) throws {
        guard let data = value.data(using: .utf8) else {
            throw KeychainError.encodingFailed
        }
        lock.lock()
        defer { lock.unlock() }

        let baseQuery: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key
        ]
        SecItemDelete(baseQuery as CFDictionary)

        var addQuery = baseQuery
        addQuery[kSecValueData as String] = data
        addQuery[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlockedThisDeviceOnly

        let status = SecItemAdd(addQuery as CFDictionary, nil)
        if status != errSecSuccess {
            throw KeychainError.unexpectedStatus(status)
        }
    }

    func loadString(forKey key: String) -> String? {
        lock.lock()
        defer { lock.unlock() }

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        guard status == errSecSuccess,
              let data = item as? Data,
              let string = String(data: data, encoding: .utf8) else {
            return nil
        }
        return string
    }

    func delete(forKey key: String) {
        lock.lock()
        defer { lock.unlock() }

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key
        ]
        SecItemDelete(query as CFDictionary)
    }
}

/// Store in-memory khusus unit test agar tidak bergantung pada entitlement Keychain simulator.
final class InMemoryKeychainStore: KeychainStoring, @unchecked Sendable {
    private var storage: [String: String] = [:]
    private let lock = NSLock()

    func saveString(_ value: String, forKey key: String) throws {
        lock.lock()
        defer { lock.unlock() }
        storage[key] = value
    }

    func loadString(forKey key: String) -> String? {
        lock.lock()
        defer { lock.unlock() }
        return storage[key]
    }

    func delete(forKey key: String) {
        lock.lock()
        defer { lock.unlock() }
        storage.removeValue(forKey: key)
    }
}
