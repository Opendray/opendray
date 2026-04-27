package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type healthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	UptimeSec int64  `json:"uptime_s"`
	DBOK      bool   `json:"db_ok"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbOK := true
	if s.deps.DB != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.deps.DB.Ping(ctx); err != nil {
			dbOK = false
		}
	}

	resp := healthResponse{
		Status:    "ok",
		Version:   s.deps.Version.Version,
		Commit:    s.deps.Version.Commit,
		UptimeSec: int64(time.Since(s.deps.StartedAt).Seconds()),
		DBOK:      dbOK,
	}
	if !dbOK {
		resp.Status = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
