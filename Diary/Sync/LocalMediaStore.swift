import Foundation

struct LocalMediaStore {
    /// Optional override for the storage root. Defaults to the app's
    /// Application Support directory; tests inject a temporary directory.
    private let overrideRoot: URL?

    init(root: URL? = nil) {
        self.overrideRoot = root
    }

    func relativePath(attachmentID: String, filename: String) -> String {
        let safeFilename = filename
            .replacingOccurrences(of: "/", with: "-")
            .replacingOccurrences(of: ":", with: "-")
        return "\(attachmentID)/\(safeFilename)"
    }

    func fileURL(relativePath: String) throws -> URL {
        try rootURL().appending(path: relativePath)
    }

    func fileExists(relativePath: String) -> Bool {
        guard let url = try? fileURL(relativePath: relativePath) else {
            return false
        }

        return FileManager.default.fileExists(atPath: url.path())
    }

    func save(data: Data, relativePath: String) throws {
        let url = try fileURL(relativePath: relativePath)
        try FileManager.default.createDirectory(at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        try data.write(to: url, options: [.atomic])
    }

    /// Removes the cached media for an attachment. Files live under
    /// `<attachmentID>/<filename>`, so deleting the attachment's directory
    /// reclaims the file regardless of its name. Best-effort: a missing
    /// directory is not an error.
    func removeAttachment(attachmentID: String) {
        guard let root = try? rootURL() else { return }
        let directory = root.appending(path: attachmentID, directoryHint: .isDirectory)
        try? FileManager.default.removeItem(at: directory)
    }

    private func rootURL() throws -> URL {
        if let overrideRoot {
            try FileManager.default.createDirectory(at: overrideRoot, withIntermediateDirectories: true)
            return overrideRoot
        }

        let baseURL = try FileManager.default.url(
            for: .applicationSupportDirectory,
            in: .userDomainMask,
            appropriateFor: nil,
            create: true
        )
        let url = baseURL.appending(path: "Media", directoryHint: .isDirectory)
        try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
        return url
    }
}
