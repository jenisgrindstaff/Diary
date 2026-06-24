package app

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"diary/server/internal/diary"
	"diary/server/internal/store"
)

type Server struct {
	cfg       Config
	logger    *slog.Logger
	db        *sql.DB
	store     *store.Store
	importer  *diary.Importer
	statusMu  sync.RWMutex
	reindex   ReindexResult
	startedAt time.Time
}

type ReindexResult struct {
	StartedAt                time.Time `json:"started_at"`
	FinishedAt               time.Time `json:"finished_at"`
	EntryCount               int       `json:"entry_count"`
	TombstoneCount           int       `json:"tombstone_count"`
	BackfilledSubjectDetails int       `json:"backfilled_subject_details"`
	Error                    string    `json:"error,omitempty"`
}

func New(cfg Config, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}

	for _, dir := range []string{cfg.VaultDir, cfg.ImportDir, cfg.DataDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	for _, dir := range []string{
		filepath.Join(cfg.VaultDir, "entries"),
		filepath.Join(cfg.VaultDir, "assets"),
		filepath.Join(cfg.VaultDir, "deletions"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	db, err := store.Open(cfg.DBPath())
	if err != nil {
		return nil, err
	}

	st := store.New(db)
	if err := st.Migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	importer := diary.NewImporter(cfg.VaultDir, cfg.ImportDir)
	server := &Server{
		cfg:       cfg,
		logger:    logger,
		db:        db,
		store:     st,
		importer:  importer,
		startedAt: time.Now().UTC(),
	}

	if err := server.Reindex(); err != nil {
		_ = db.Close()
		return nil, err
	}
	result := server.lastReindexResult()
	logger.Info(
		"diary server initialized",
		"vault", cfg.VaultDir,
		"data", cfg.DataDir,
		"imports", cfg.ImportDir,
		"entries", result.EntryCount,
		"tombstones", result.TombstoneCount,
		"backfilled_subject_details", result.BackfilledSubjectDetails,
	)

	return server, nil
}

func (s *Server) Close() error {
	if s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Server) Reindex() error {
	result := ReindexResult{StartedAt: time.Now().UTC()}
	entries, stats, err := diary.ReadVaultWithStats(s.cfg.VaultDir)
	if err != nil {
		result.FinishedAt = time.Now().UTC()
		result.Error = err.Error()
		s.setReindexResult(result)
		return err
	}
	tombstones, err := diary.ReadTombstones(s.cfg.VaultDir)
	if err != nil {
		result.FinishedAt = time.Now().UTC()
		result.EntryCount = len(entries)
		result.BackfilledSubjectDetails = stats.BackfilledSubjectDetails
		result.Error = err.Error()
		s.setReindexResult(result)
		return err
	}

	result.EntryCount = len(entries)
	result.TombstoneCount = len(tombstones)
	result.BackfilledSubjectDetails = stats.BackfilledSubjectDetails
	if err := s.store.ReplaceIndex(entries, tombstones); err != nil {
		result.FinishedAt = time.Now().UTC()
		result.Error = err.Error()
		s.setReindexResult(result)
		return err
	}

	result.FinishedAt = time.Now().UTC()
	s.setReindexResult(result)
	s.logger.Info(
		"diary vault reindexed",
		"vault", s.cfg.VaultDir,
		"entries", result.EntryCount,
		"tombstones", result.TombstoneCount,
		"backfilled_subject_details", result.BackfilledSubjectDetails,
	)
	if result.EntryCount == 0 {
		s.logger.Warn("diary vault has no indexed entries", "vault", s.cfg.VaultDir)
	}
	return nil
}

func (s *Server) setReindexResult(result ReindexResult) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	s.reindex = result
}

func (s *Server) lastReindexResult() ReindexResult {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()
	return s.reindex
}

// indexEntry re-reads a single entry file from disk and upserts it into the
// index. Used by the per-entry write paths instead of a full Reindex, so one
// mutation does O(1) work rather than re-reading the whole vault.
func (s *Server) indexEntry(path string) error {
	entry, err := diary.ReadEntry(s.cfg.VaultDir, path)
	if err != nil {
		return err
	}
	return s.store.IndexEntry(entry)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func publicMessage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, os.ErrNotExist) {
		return "not found"
	}

	return err.Error()
}
