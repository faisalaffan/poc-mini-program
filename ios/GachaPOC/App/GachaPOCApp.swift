import SwiftUI

@main
struct GachaPOCApp: App {
    @State private var container = AppContainer()

    var body: some Scene {
        WindowGroup {
            RootView(container: container)
                .preferredColorScheme(.light)
                .task {
                    _ = try? await container.tokenManager.ensureMasterSession()
                }
        }
    }
}
