基于 Go 实现的航空发动机叶片疲劳循环计数服务，一款纯后端工程分析服务，处理载荷循环计数、疲劳损伤评估与版本化结果发布。

# BENZHI 评测说明 · task224-bladecycle

本文件面向评测环境，说明构建、运行与 `--smoke-test` 契约。

## 构建与运行

```bash
# 本地冒烟
go run ./cmd/bladecycle --smoke-test

# 启动服务
go run ./cmd/bladecycle --addr :8080 --db ./bladecycle.db

# Docker 双架构构建
bash build_benzhi_docker.sh <镜像名> <平台>   # 平台如 linux/amd64、linux/arm64
```

## --smoke-test 契约

`--smoke-test` 不启动长驻服务，而是：

1. 打开数据库 A，走完整闭环：材料 → 试验 → 采集 → 多通道遥测（含漂移段、
   幂等重放、缺口）→ 分析（雨流计数 + Miner 损伤）→ 快照 → 确认 → 封存；
2. 幂等验证：重复 seq_no 被跳过；
3. 漂移段被排除出计数；
4. 封存后拒绝再接收；
5. 关闭数据库 A，重开同一路径数据库 B，验证试验 sealed、循环数、快照版本与
   损伤值一致恢复。

全部通过后输出 `SMOKE OK: ...` 并以 **退出码 0** 结束；任一失败输出
`SMOKE TEST FAILED: <原因>` 并以退出码 1 结束。

## 入口契约

- `--addr :8080` 监听地址（默认 `:8080`）
- `--db ./bladecycle.db` SQLite 数据库路径（默认 `./bladecycle.db`）
- `--smoke-test` 执行端到端冒烟并退出

## Docker 契约

- 基础镜像 `docker.m.daocloud.io/library/golang:1.26.3-bookworm`
- `ENV GOPROXY=https://goproxy.cn,direct`、`ENV GOSUMDB=sum.golang.google.cn`
- `CGO_ENABLED=0 go build ./cmd/bladecycle`
- `ENTRYPOINT ["/app/bladecycle"]`，`CMD ["--smoke-test"]`（运行冒烟即退出 0）
- 双架构 `linux/amd64` 与 `linux/arm64` 均通过 build + `--smoke-test`

## API 前缀

统一 `/api`，共 22 个路由，覆盖材料、试验生命周期、遥测接收/漂移标记、循环
计数、损伤查询、寿命快照与统计健康。
