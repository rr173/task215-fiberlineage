基于 Go 实现的古籍纸张纤维谱系比对全栈 Web 项目，一款文献研究分析服务，处理样本证据比对、谱系推断与版本冻结。

# fiberlineage 评测说明（BENZHI）

## 一句话

古籍纸张纤维谱系比对台：按证据权重比对古籍样本的纤维谱/染料峰/修补层序，
构建并维护来源谱系假设图，记录反例，支持版本冻结与差异查看。

## 标准命令（全部须真成功，保留真实退出码）

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/fiberlineage --smoke-test
```

## --smoke-test 契约

`--smoke-test` 不启动长驻服务，而是在临时 SQLite 上执行建表迁移并运行一次
一致性自检（`SelfCheck`），全绿则以退出码 0 结束并打印 `smoke-test OK`。
这是 Docker `CMD` 与双架构验证的唯一判据。

## Web 页面

启动服务后访问 `http://127.0.0.1:8080/`，可查看证据矩阵、样本相似度、有效证据、谱系假设和来源谱系图。
页面由服务内嵌并直接调用 `/api/samples`、`/api/evidence`、`/api/similarity` 和 `/api/lineages`。

## Docker 双架构构建

```bash
bash build_benzhi_docker.sh fiberlineage:amd64 linux/amd64
bash build_benzhi_docker.sh fiberlineage:arm64 linux/arm64
docker run --rm fiberlineage:amd64 --smoke-test
docker run --rm fiberlineage:arm64 --smoke-test
```

## HTTP API 冒烟

```bash
go run ./cmd/fiberlineage -addr :8080 -db /tmp/fl.db &
curl -s http://127.0.0.1:8080/api/health
curl -s -X POST http://127.0.0.1:8080/api/samples -d '{"code":"S1","title":"甲","source":"A"}'
curl -s -X POST http://127.0.0.1:8080/api/samples/1/detect \
  -d '{"fiber_spectrum":[{"band":1,"intensity":0.9}],"unit_known":true}'
curl -s -X POST http://127.0.0.1:8080/api/versions -d '{"code":"V1"}'
curl -s http://127.0.0.1:8080/api/self-check
```

## 核心数据表

`samples` / `detection_fiber` / `detection_dye` / `detection_repair` /
`evidence` / `similarity_edges` / `lineage_hypotheses` / `lineage_edges` /
`counterexamples` / `research_versions`
