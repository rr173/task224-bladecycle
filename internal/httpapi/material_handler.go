package httpapi

import "net/http"

type createMaterialReq struct {
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	FatigueStrengthCoef float64 `json:"fatigue_strength_coef"`
	FatigueExponent     float64 `json:"fatigue_exponent"`
	UltimateStrength    float64 `json:"ultimate_strength"`
	MeanStressMethod    string  `json:"mean_stress_method"`
}

func (s *Server) createMaterial(w http.ResponseWriter, r *http.Request) {
	var req createMaterialReq
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}
	m, err := s.app.Mats.Create(req.Code, req.Name, req.FatigueStrengthCoef, req.FatigueExponent, req.UltimateStrength, req.MeanStressMethod)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) listMaterials(w http.ResponseWriter, r *http.Request) {
	ms, err := s.app.Mats.List()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ms)
}

func (s *Server) getMaterial(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	m, err := s.app.Mats.Get(id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}
