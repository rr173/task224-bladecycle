package httpapi

import "net/http"

type createTrialReq struct {
	Name            string  `json:"name"`
	EngineModel     string  `json:"engine_model"`
	MaterialBatchID int64   `json:"material_batch_id"`
	SampleRateHz    float64 `json:"sample_rate_hz"`
}

func (s *Server) createTrial(w http.ResponseWriter, r *http.Request) {
	var req createTrialReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}
	t, err := s.app.Trials.Create(req.Name, req.EngineModel, req.MaterialBatchID, req.SampleRateHz)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) listTrials(w http.ResponseWriter, r *http.Request) {
	ts, err := s.app.Trials.List()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ts)
}

func (s *Server) getTrial(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	t, err := s.app.Trials.Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) startTrial(w http.ResponseWriter, r *http.Request) {
	s.transitionTrial(w, r, func(id int64) (interface{}, error) {
		return s.app.Trials.Start(id)
	})
}

func (s *Server) finishTrial(w http.ResponseWriter, r *http.Request) {
	s.transitionTrial(w, r, func(id int64) (interface{}, error) {
		return s.app.Trials.FinishAcquisition(id)
	})
}

func (s *Server) confirmTrial(w http.ResponseWriter, r *http.Request) {
	s.transitionTrial(w, r, func(id int64) (interface{}, error) {
		return s.app.Trials.Confirm(id)
	})
}

func (s *Server) sealTrial(w http.ResponseWriter, r *http.Request) {
	s.transitionTrial(w, r, func(id int64) (interface{}, error) {
		return s.app.Trials.Seal(id)
	})
}

type transitionFn func(id int64) (interface{}, error)

func (s *Server) transitionTrial(w http.ResponseWriter, r *http.Request, fn transitionFn) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	t, err := fn(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}
