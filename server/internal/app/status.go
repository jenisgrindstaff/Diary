package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"diary/server/internal/diary"
)

type AdminStatus struct {
	StartedAt   time.Time     `json:"started_at"`
	Paths       StatusPaths   `json:"paths"`
	Counts      StatusCounts  `json:"counts"`
	People      PeopleStatus  `json:"people"`
	Auth        AuthStatus    `json:"auth"`
	LastReindex ReindexResult `json:"last_reindex"`
}

type StatusPaths struct {
	VaultDir  string `json:"vault_dir"`
	DataDir   string `json:"data_dir"`
	ImportDir string `json:"import_dir"`
	DBPath    string `json:"db_path"`
}

type StatusCounts struct {
	Entries    int `json:"entries"`
	Assets     int `json:"assets"`
	Tombstones int `json:"tombstones"`
}

type PeopleStatus struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Count   int    `json:"count"`
	Message string `json:"message,omitempty"`
}

type AuthStatus struct {
	APIEnabled        bool   `json:"api_enabled"`
	WebProxyEnabled   bool   `json:"web_proxy_enabled"`
	WebAuthHeader     string `json:"web_auth_header,omitempty"`
	WebProxySecretSet bool   `json:"web_proxy_secret_set"`
}

func (s *Server) adminStatus() AdminStatus {
	entryCount, _ := s.store.EntryCount()
	assetCount, _ := s.store.AttachmentCount()
	tombstoneCount, _ := s.store.TombstoneCount()

	return AdminStatus{
		StartedAt: s.startedAt,
		Paths: StatusPaths{
			VaultDir:  s.cfg.VaultDir,
			DataDir:   s.cfg.DataDir,
			ImportDir: s.cfg.ImportDir,
			DBPath:    s.cfg.DBPath(),
		},
		Counts: StatusCounts{
			Entries:    entryCount,
			Assets:     assetCount,
			Tombstones: tombstoneCount,
		},
		People:      s.peopleStatus(),
		Auth:        s.authStatus(),
		LastReindex: s.lastReindexResult(),
	}
}

func (s *Server) peopleStatus() PeopleStatus {
	path := filepath.Join(s.cfg.VaultDir, "people.yaml")
	status := PeopleStatus{Path: path}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			status.Message = "people.yaml not found"
			return status
		}
		status.Message = err.Error()
		return status
	}

	status.Exists = true
	people, err := diary.LoadPeople(s.cfg.VaultDir)
	if err != nil {
		status.Message = err.Error()
		return status
	}
	status.Count = len(people)
	return status
}

func (s *Server) authStatus() AuthStatus {
	return AuthStatus{
		APIEnabled:        strings.TrimSpace(s.cfg.APIToken) != "",
		WebProxyEnabled:   strings.TrimSpace(s.cfg.WebAuthHeader) != "" || strings.TrimSpace(s.cfg.WebAuthProxySecret) != "",
		WebAuthHeader:     s.cfg.WebAuthHeader,
		WebProxySecretSet: strings.TrimSpace(s.cfg.WebAuthProxySecret) != "",
	}
}

func (s *Server) handleAdminStatusPage(w http.ResponseWriter, r *http.Request) {
	data := adminStatusPageData{
		Status:    s.adminStatus(),
		CSRFToken: ensureCSRFToken(w, r),
	}
	if err := pageTemplate.ExecuteTemplate(w, "adminStatus", data); err != nil {
		s.logger.Error("render admin status failed", "error", err)
	}
}

func (s *Server) handleAdminStatusAPI(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.adminStatus())
}

type adminStatusPageData struct {
	Status    AdminStatus
	CSRFToken string
	Public    bool
}
