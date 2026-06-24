package store

import (
	"fmt"
	"strings"

	"diary/server/internal/diary"
)

type EntryListOptions struct {
	Query  string
	Year   string
	Month  string
	Limit  int
	Offset int
}

type ArchiveCount struct {
	Year  string
	Month string
	Count int
}

func (s *Store) EntriesPage(opts EntryListOptions) ([]diary.Entry, int, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	joinClause := ""
	whereParts := []string{}
	args := []any{}
	if query := FTSQuery(opts.Query); query != "" {
		joinClause = "JOIN entries_fts ON entries_fts.id = e.id"
		whereParts = append(whereParts, "entries_fts MATCH ?")
		args = append(args, query)
	}

	year, month, validDateFilter := cleanDateFilter(opts.Year, opts.Month)
	if validDateFilter && year != "" {
		whereParts = append(whereParts, "substr(e.created_at, 1, 4) = ?")
		args = append(args, year)
	}
	if validDateFilter && month != "" {
		whereParts = append(whereParts, "substr(e.created_at, 6, 2) = ?")
		args = append(args, month)
	}
	if !validDateFilter {
		whereParts = append(whereParts, "1 = 0")
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	countArgs := append([]any{}, args...)
	var total int
	if err := s.db.QueryRow(`
SELECT COUNT(*)
FROM entries e
`+joinClause+`
`+whereClause, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	pageArgs := append([]any{}, args...)
	pageArgs = append(pageArgs, opts.Limit, opts.Offset)
	rows, err := s.db.Query(`
SELECT e.id, e.created_at, e.updated_at, e.server_revision, e.title, e.excerpt, e.body_markdown, e.source_path, e.vault_path, e.tags_json, e.people_json, e.subject_details_json, e.context_json
FROM entries e
`+joinClause+`
`+whereClause+`
ORDER BY e.created_at DESC, e.title ASC
LIMIT ? OFFSET ?`, pageArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries, err := scanEntries(rows, s.attachmentsForEntries)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func (s *Store) ArchiveCounts() ([]ArchiveCount, error) {
	rows, err := s.db.Query(`
SELECT substr(created_at, 1, 4), substr(created_at, 6, 2), COUNT(*)
FROM entries
GROUP BY substr(created_at, 1, 4), substr(created_at, 6, 2)
ORDER BY substr(created_at, 1, 4) DESC, substr(created_at, 6, 2) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []ArchiveCount
	for rows.Next() {
		var count ArchiveCount
		if err := rows.Scan(&count.Year, &count.Month, &count.Count); err != nil {
			return nil, err
		}
		counts = append(counts, count)
	}
	return counts, rows.Err()
}

func (s *Store) EntryCount() (int, error) {
	return s.countRows("entries")
}

func (s *Store) AttachmentCount() (int, error) {
	return s.countRows("attachments")
}

func (s *Store) TombstoneCount() (int, error) {
	return s.countRows("tombstones")
}

func (s *Store) countRows(table string) (int, error) {
	if !identifierPattern.MatchString(table) {
		return 0, fmt.Errorf("invalid identifier: table=%q", table)
	}

	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count)
	return count, err
}

func cleanDateFilter(year string, month string) (string, string, bool) {
	year = strings.TrimSpace(year)
	month = strings.TrimSpace(month)
	if year == "" && month == "" {
		return "", "", true
	}
	if len(year) != 4 || !allDigits(year) {
		return "", "", false
	}

	if month == "" {
		return year, "", true
	}
	if len(month) == 1 {
		month = "0" + month
	}
	if len(month) != 2 || !allDigits(month) || month < "01" || month > "12" {
		return "", "", false
	}
	return year, month, true
}

func allDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}
