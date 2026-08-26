基于 Go 实现的数据仓库血缘列级截断诊断服务，一款纯后端分析服务，接收多表版本元数据与列级变换声明，构建列级血缘图谱，定位上游列重命名导致的派生列截断，并发布可追溯诊断结论。

# linagediag 评测说明

数据仓库血缘列级截断诊断服务（纯后端）。

## 本地构建与验证

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build -o linagediag ./cmd/linagediag
go test ./...
go vet ./...
./linagediag --smoke-test --db smoke.db
./linagediag --addr :8080 --db linagediag.db
```

- `--addr`：HTTP 监听地址，默认 `:8080`（Dockerfile 不声明端口）
- `--db`：SQLite 数据库文件路径
- `--smoke-test`：执行端到端场景后关闭并重新打开数据库验证持久化与重启恢复，最后以 0 退出码结束

## 冒烟自测契约（--smoke-test）

`--smoke-test` 是 Docker `CMD` 与双架构验证的唯一判据：真实创建批次与多版本表/列元数据、登记列级变换声明、构建血缘图谱、触发截断诊断、按重命名推断修订断裂边或豁免、发布快照并封存批次，关闭并重开同一数据库验证持久化与重启恢复，最终以退出码 0 结束。容器里只传 flag，不传二进制路径位置参数。

## Docker 构建与双架构验证

使用项目提供的 `build_benzhi_docker.sh` 构建评测镜像；Dockerfile 不声明端口，服务监听地址由运行参数 `--addr` 指定。

```bash
./build_benzhi_docker.sh task249-linagediag:amd64 linux/amd64
docker run --rm task249-linagediag:amd64 --smoke-test

./build_benzhi_docker.sh task249-linagediag:arm64 linux/arm64
docker run --rm task249-linagediag:arm64 --smoke-test

docker run --rm -P task249-linagediag:amd64 --addr :8080 --db ./app.db
```

两项 `docker run --smoke-test` 均须退出码 0。

## 主要 API（前缀 /api）

- 批次：`POST /api/batches`、`GET /api/batches`、`GET /api/batches/{id}`
- 元数据：`POST /api/batches/{id}/tables`、`GET /api/batches/{id}/tables`、
  `GET /api/batches/{id}/columns?table_id=`
- 变换/作业：`POST /api/batches/{id}/transforms`、`POST /api/batches/{id}/jobs`
- 构图/诊断：`POST /api/batches/{id}/build`、`GET /api/batches/{id}/diagnose`、
  `GET /api/batches/{id}/lineage`、`GET /api/batches/{id}/impact?table=&column=`
- 裁决：`POST /api/batches/{id}/edges/{eid}/confirm`、`POST /api/batches/{id}/edges/{eid}/exempt`、
  `POST /api/batches/{id}/edges/{eid}/revise`、`GET /api/batches/{id}/edges`、
  `GET /api/batches/{id}/adjudications`
- 快照/封存：`POST /api/batches/{id}/snapshots`、`GET /api/batches/{id}/snapshots`、
  `POST /api/batches/{id}/seal`、`POST /api/batches/{id}/confirm-publish`
- 闭环/健康：`POST /api/batches/{id}/scenario`、`GET /api/health`

## 环境

- Go 1.26.3，`CGO_ENABLED=0`，`GOPROXY=https://goproxy.cn,direct`，`GOSUMDB=sum.golang.google.cn`
- SQLite 驱动 modernc.org/sqlite v1.52.0（纯 Go），SQLite 3.46.1
