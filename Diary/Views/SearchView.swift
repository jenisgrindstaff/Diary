import SwiftData
import SwiftUI

struct SearchView: View {
    @Environment(\.modelContext) private var modelContext
    @Environment(AppState.self) private var appState

    @Query(
        filter: #Predicate<DiaryEntry> { $0.isTombstoned == false },
        sort: \DiaryEntry.createdAt,
        order: .reverse
    )
    private var entries: [DiaryEntry]
    @Query(sort: \PendingChange.createdAt, order: .forward) private var pendingChanges: [PendingChange]

    @State private var syncCoordinator = SyncCoordinator()
    @State private var searchText = ""
    @State private var serverSearchState = ServerSearchState.idle

    private var pendingByEntryID: [String: PendingChange] {
        Dictionary(pendingChanges.map { ($0.entryID, $0) }, uniquingKeysWith: { first, _ in first })
    }

    private var results: [DiaryEntry] {
        guard !searchTerms.isEmpty else {
            return entries
        }

        return entries.filter { entry in
            searchTerms.allSatisfy { entry.searchTextStorage.contains($0) }
        }
    }

    private var trimmedSearchText: String {
        searchText.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var searchTerms: [String] {
        trimmedSearchText
            .folding(options: [.caseInsensitive, .diacriticInsensitive], locale: .current)
            .split(whereSeparator: \.isWhitespace)
            .map(String.init)
    }

    var body: some View {
        NavigationStack {
            List {
                if !trimmedSearchText.isEmpty && !results.isEmpty {
                    SearchControlRow(
                        localResultCount: results.count,
                        totalCount: entries.count,
                        state: serverSearchState,
                        isDisabled: syncCoordinator.isSyncing
                    ) {
                        Task {
                            await searchServer()
                        }
                    }
                } else if !trimmedSearchText.isEmpty {
                    SearchControlRow(
                        localResultCount: 0,
                        totalCount: entries.count,
                        state: serverSearchState,
                        isDisabled: syncCoordinator.isSyncing
                    ) {
                        Task {
                            await searchServer()
                        }
                    }
                }

                ForEach(results) { entry in
                    NavigationLink(value: entry.id) {
                        EntryRow(entry: entry, pendingChange: pendingByEntryID[entry.id])
                    }
                    .listRowInsets(EdgeInsets(top: 10, leading: 18, bottom: 10, trailing: 18))
                }
            }
            .listStyle(.plain)
            .overlay {
                if entries.isEmpty && trimmedSearchText.isEmpty {
                    ContentUnavailableView(
                        "No Entries",
                        systemImage: "book.closed",
                        description: Text("Sync with your Markdown diary server to fill the offline cache.")
                    )
                } else if results.isEmpty {
                    ContentUnavailableView.search(text: searchText)
                }
            }
            .navigationTitle("Search")
            .navigationDestination(for: String.self) { entryID in
                EntryDetailResolver(entryID: entryID)
            }
            .searchable(text: $searchText, prompt: "Entries, tags, people")
            .searchToolbarBehavior(.minimize)
            .onChange(of: trimmedSearchText) { _, _ in
                serverSearchState = .idle
            }
        }
    }

    private func searchServer() async {
        guard !trimmedSearchText.isEmpty else {
            return
        }

        serverSearchState = .searching

        do {
            let count = try await syncCoordinator.searchServer(
                query: trimmedSearchText,
                modelContext: modelContext,
                appState: appState
            )
            serverSearchState = .completed(count)
        } catch {
            serverSearchState = .failed(error.localizedDescription)
        }
    }
}

private enum ServerSearchState: Equatable {
    case idle
    case searching
    case completed(Int)
    case failed(String)
}

private struct SearchControlRow: View {
    let localResultCount: Int
    let totalCount: Int
    let state: ServerSearchState
    let isDisabled: Bool
    let searchServer: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Label("\(localResultCount) local result\(localResultCount == 1 ? "" : "s")", systemImage: "magnifyingglass")
                Spacer()
                Text("\(totalCount) cached")
                    .foregroundStyle(.secondary)
            }

            HStack {
                serverStatus
                Spacer()
                Button("Search Server", systemImage: "icloud.and.arrow.down") {
                    searchServer()
                }
                .disabled(isDisabled || state == .searching)
            }
        }
        .font(.footnote)
        .foregroundStyle(.secondary)
        .listRowSeparator(.hidden)
    }

    @ViewBuilder
    private var serverStatus: some View {
        switch state {
        case .idle:
            Text("Server search checks canonical Markdown.")
        case .searching:
            Label("Searching server", systemImage: "hourglass")
        case .completed(let count):
            Label("\(count) server result\(count == 1 ? "" : "s") imported", systemImage: "checkmark.circle")
                .foregroundStyle(.green)
        case .failed(let message):
            Label(message, systemImage: "exclamationmark.triangle.fill")
                .foregroundStyle(.red)
        }
    }
}

#Preview {
    SearchView()
        .modelContainer(PreviewData.container)
}
