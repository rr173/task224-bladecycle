# task224-bladecycle · 航空发动机叶片疲劳循环计数服务

从叶片应变遥测中提取疲劳循环，按材料批次累计 Miner 损伤，冻结寿命快照。

## 业务

- 雨流循环计数（rainflow counting，四点法）
- Miner 线性累积损伤（结合材料 Basquin S-N 曲线与 Goodman/Morrow 平均应力修正）
- 遥测缺口（seq_no 跳号）与传感器漂移（线性趋势）自动检测
- 寿命快照版本冻结

## 运行

```bash
# 端到端冒烟测试（真实建库→计数→损伤→快照→关闭重开验证恢复）
go run ./cmd/bladecycle --smoke-test

# 启动 HTTP 服务
go run ./cmd/bladecycle --addr :8080 --db ./bladecycle.db
```

## 构建门禁

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/bladecycle --smoke-test
```

## API 入口（统一 /api 前缀）

- 材料：`POST/GET /api/materials`、`GET /api/materials/{id}`
- 试验：`POST/GET /api/trials`、`GET /api/trials/{id}`、
  `POST /api/trials/{id}/start|finish|confirm|seal`
- 遥测：`POST/GET /api/trials/{id}/telemetry`、`POST /api/telemetry/{id}/drift`
- 分析：`POST /api/trials/{id}/analyze`、`GET /api/trials/{id}/cycles`、
  `GET /api/trials/{id}/damage`、`GET /api/trials/{id}/damage/history`
- 快照：`POST /api/trials/{id}/snapshots`、`GET /api/trials/{id}/snapshots`、
  `GET /api/snapshots/{id}`
- 统计/健康：`GET /api/stats`、`GET /api/health`

## 持久化

SQLite（`modernc.org/sqlite`，纯 Go 驱动，CGO 无关）。7 张表：trials、materials、
channels、telemetry_segments、cycles、damage_records、snapshots。
