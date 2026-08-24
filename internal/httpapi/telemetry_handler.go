package httpapi

import "net/http"

type ingestTelemetryReq struct {
	ChannelIndex int       `json:"channel_index"`
	SensorID     string    `json:"sensor_id"`
	SeqNo        int64     `json:"seq_no"`
	RPM          float64   `json:"rpm"`
	Temperature  float64   `json:"temperature"`
	Strain       []float64 `json:"strain"`
}

func (s *Server) ingestTelemetry(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	trial, err := s.app.Trials.EnsureWritable(id)
	if err != nil {
		writeError(w, err)
		return
	}
	var req ingestTelemetryReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}
	res, err := s.app.Tele.Ingest(trial, req.ChannelIndex, req.SensorID, req.SeqNo, req.RPM, req.Temperature, req.Strain)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) listTelemetry(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	segs, err := s.app.Tele.ListByTrial(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, segs)
}

type markDriftReq struct {
	Reason string `json:"reason"`
}

func (s *Server) markDrift(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req markDriftReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}
	if err := s.app.Tele.MarkDrift(id, req.Reason); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"segment_id": id, "status": "drift"})
}
