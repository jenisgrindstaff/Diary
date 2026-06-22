import SwiftData
import SwiftUI

struct SearchView: View {
    @Query(
        filter: #Predicate<DiaryEntry> { $0.isTombstoned == false },
        sort: \DiaryEntry.createdAt,
        order: .reverse
    )
    private var entries: [DiaryEntry]
    @Query(sort: \PendingChange.createdAt, order: .forward) private var pendingChanges: [PendingChange]

    @State private var searchText = ""

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
                    SearchResultSummary(resultCount: results.count, totalCount: entries.count)
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
        }
    }
}

private struct SearchResultSummary: View {
    let resultCount: Int
    let totalCount: Int

    var body: some View {
        HStack {
            Label("\(resultCount) result\(resultCount == 1 ? "" : "s")", systemImage: "magnifyingglass")
            Spacer()
            Text("\(totalCount) entries")
                .foregroundStyle(.secondary)
        }
        .font(.footnote)
        .foregroundStyle(.secondary)
        .accessibilityElement(children: .combine)
        .listRowSeparator(.hidden)
    }
}

#Preview {
    SearchView()
        .modelContainer(PreviewData.container)
}
