import Foundation

enum SDUIValidationError: Error, Sendable, Equatable {
    case unsupportedSchemaVersion(received: Int, maxSupported: Int)
    case emptyComponents
    case unknownComponent(nodeId: String, typeName: String)
    case disallowedAction(nodeId: String, actionName: String)
    case malformedJSON(String)

    var reasonText: String {
        switch self {
        case let .unsupportedSchemaVersion(received, maxSupported):
            return "Schema v\(received) tidak didukung oleh app ini (maksimal v\(maxSupported)). Fallback ke layar native bawaan aktif."
        case .emptyComponents:
            return "Payload SDUI tidak memiliki komponen untuk dirender."
        case let .unknownComponent(nodeId, typeName):
            return "Komponen '\(typeName)' pada node '\(nodeId)' tidak dikenal dan ditolak renderer."
        case let .disallowedAction(nodeId, actionName):
            return "Aksi '\(actionName)' pada node '\(nodeId)' tidak ada di whitelist native."
        case let .malformedJSON(detail):
            return "Format JSON SDUI tidak valid: \(detail)"
        }
    }
}

struct SDUIValidator: Sendable {
    static let supportedSchemaVersion = 1

    static func validate(jsonData: Data) -> Result<ValidatedSDUIScreen, SDUIValidationError> {
        do {
            let payload = try JSONDecoder().decode(SDUIScreenPayload.self, from: jsonData)
            return validate(payload: payload)
        } catch {
            return .failure(.malformedJSON(error.localizedDescription))
        }
    }

    static func validate(payload: SDUIScreenPayload) -> Result<ValidatedSDUIScreen, SDUIValidationError> {
        guard payload.schemaVersion >= 1 && payload.schemaVersion <= supportedSchemaVersion else {
            return .failure(.unsupportedSchemaVersion(
                received: payload.schemaVersion,
                maxSupported: supportedSchemaVersion
            ))
        }

        guard !payload.components.isEmpty else {
            return .failure(.emptyComponents)
        }

        var validatedNodes: [ValidatedSDUINode] = []
        validatedNodes.reserveCapacity(payload.components.count)

        for node in payload.components {
            guard let compType = SDUIComponentType(rawValue: node.type) else {
                return .failure(.unknownComponent(nodeId: node.id, typeName: node.type))
            }

            var resolvedAction: SDUIActionType?
            if let rawAction = node.action {
                guard let allowed = SDUIActionType(rawValue: rawAction) else {
                    return .failure(.disallowedAction(nodeId: node.id, actionName: rawAction))
                }
                resolvedAction = allowed
            }

            validatedNodes.append(
                ValidatedSDUINode(
                    id: node.id,
                    type: compType,
                    props: node.props ?? [:],
                    action: resolvedAction
                )
            )
        }

        return .success(
            ValidatedSDUIScreen(
                schemaVersion: payload.schemaVersion,
                screenId: payload.screenId,
                variant: payload.variant,
                navigationTitle: payload.navigationTitle,
                backgroundAsset: payload.backgroundAsset,
                tableVersion: payload.tableVersion,
                nodes: validatedNodes
            )
        )
    }
}
