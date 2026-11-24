import Foundation

@MainActor
final class AppContainer {
    let config: AppConfig
    let keychain: KeychainStore
    let pinner: CertificatePinner
    let tokenManager: TokenManager
    let apiClient: APIClient
    let metrics: MetricsCollector
    let nativeVM: NativeGachaViewModel
    let sduiVM: SDUIGachaViewModel

    init() {
        let cfg = AppConfig()
        let kc = KeychainStore()
        #if DEBUG
        let allowDebugPlaintext = true
        #else
        let allowDebugPlaintext = false
        #endif
        let pin = CertificatePinner(
            allowedHosts: cfg.allowedHosts,
            pinnedPublicKeyHashes: cfg.pinnedSPKIHashes,
            allowLocalDebugPlaintext: allowDebugPlaintext
        )
        let tm = TokenManager(keychain: kc, baseURL: cfg.baseURL, pinner: pin)
        let api = APIClient(baseURL: cfg.baseURL, tokenManager: tm, pinner: pin)
        let col = MetricsCollector()

        self.config = cfg
        self.keychain = kc
        self.pinner = pin
        self.tokenManager = tm
        self.apiClient = api
        self.metrics = col
        self.nativeVM = NativeGachaViewModel(api: api, metrics: col)
        self.sduiVM = SDUIGachaViewModel(api: api, metrics: col, config: cfg)
    }
}
