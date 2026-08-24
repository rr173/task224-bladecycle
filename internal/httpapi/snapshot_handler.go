package httpapi

import "net/http"

func (s *Server) publishSnapshot(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	snap, err := s.app.Snaps.Publish(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, snap)
}

func (s *Server) listSnapshots(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	snaps, err := s.app.Snaps.ListByTrial(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snaps)
}

func (s *Server) getSnapshot(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	snap, err := s.app.Snaps.Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Stats.Collect()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
