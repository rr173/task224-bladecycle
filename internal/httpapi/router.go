package httpapi

import (
	"net/http"

	"task224-bladecycle/internal/service"
)

// New 构造 HTTP 处理器，注册全部 /api 路由。
func New(app *service.App) *Server {
	s := &Server{app: app}
	return s
}

// Server 持有 App 并暴露 Handler。
type Server struct {
	app *service.App
}

// Handler 返回注册好路由的 http.Handler。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// 材料批次。
	mux.HandleFunc("POST /api/materials", s.createMaterial)
	mux.HandleFunc("GET /api/materials", s.listMaterials)
	mux.HandleFunc("GET /api/materials/{id}", s.getMaterial)

	// 试验架次。
	mux.HandleFunc("POST /api/trials", s.createTrial)
	mux.HandleFunc("GET /api/trials", s.listTrials)
	mux.HandleFunc("GET /api/trials/{id}", s.getTrial)
	mux.HandleFunc("POST /api/trials/{id}/start", s.startTrial)
	mux.HandleFunc("POST /api/trials/{id}/finish", s.finishTrial)
	mux.HandleFunc("POST /api/trials/{id}/confirm", s.confirmTrial)
	mux.HandleFunc("POST /api/trials/{id}/seal", s.sealTrial)

	// 遥测接收与漂移标记。
	mux.HandleFunc("POST /api/trials/{id}/telemetry", s.ingestTelemetry)
	mux.HandleFunc("GET /api/trials/{id}/telemetry", s.listTelemetry)
	mux.HandleFunc("POST /api/telemetry/{id}/drift", s.markDrift)

	// 循环计数与损伤。
	mux.HandleFunc("POST /api/trials/{id}/analyze", s.analyzeTrial)
	mux.HandleFunc("GET /api/trials/{id}/cycles", s.listCycles)
	mux.HandleFunc("GET /api/trials/{id}/damage", s.latestDamage)
	mux.HandleFunc("GET /api/trials/{id}/damage/history", s.damageHistory)

	// 寿命快照。
	mux.HandleFunc("POST /api/trials/{id}/snapshots", s.publishSnapshot)
	mux.HandleFunc("GET /api/trials/{id}/snapshots", s.listSnapshots)
	mux.HandleFunc("GET /api/snapshots/{id}", s.getSnapshot)

	// 统计与健康。
	mux.HandleFunc("GET /api/stats", s.stats)
	mux.HandleFunc("GET /api/health", s.health)

	return mux
}
