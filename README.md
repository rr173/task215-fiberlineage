# fiberlineage — 古籍纸张纤维谱系比对台

面向纸本文献研究者的证据比对服务：登记古籍样本的纤维谱、染料峰与修补层序，
按证据权重计算样本间相似度并构建「来源谱系假设图」，支持记录文献反例、
合并/拆分谱系、确认/否决假设，并以研究版本冻结 + 差异比对的方式沉淀定本。

## 业务闭环

1. 登记文献样本（`registered`），提交检测谱后推进到 `comparable`。
2. 在两个样本间添加证据项（纤维/染料/修补/综合），校验为 `valid` / `conflict` / `excluded`。
3. 比对引擎按证据权重合成两样本相似度，落库相似度边。
4. 在研究版本（`editing`）下建立谱系假设，加边（ancestor/descendant/same_source），
   记录反例，合并或拆分谱系。
5. 确认/否决假设；版本冻结（`frozen`）后证据不可变，可创建替代版本（`superseded`）。

## 技术栈与结构

- Go 1.26.3，纯 Go SQLite 驱动 `modernc.org/sqlite`（CGO 无关，离线可构建）。
- 分层：`internal/model`（实体/枚举）、`internal/store`（SQLite 迁移 + CRUD）、
  `internal/spectrum`（谱校验）、`internal/compare`（相似度）、`internal/evidence`（证据状态机）、
  `internal/lineage`（图环检测/合并）、`internal/version`（快照差异）、
  `internal/service`（编排）、`internal/httpapi`（HTTP + 嵌入式研究页面）。

## Web 研究页面

服务根路径 `/` 提供单文件嵌入式研究页面：读取样本、证据、相似度和谱系 API，展示证据矩阵、
有效证据摘要、谱系假设列表和来源谱系关系图。页面不引入 Node/npm 依赖，随 Go 二进制一起发布。

## 构建与验证

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/fiberlineage --smoke-test
```

## 启动

```bash
go run ./cmd/fiberlineage -addr :8080 -db fiberlineage.db
```

## API 入口（前缀 `/api`）

- 样本：`POST /api/samples`、`GET /api/samples`、`GET/PUT /api/samples/{id}`、
  `POST /api/samples/{id}/detect`、`GET /api/samples/{id}/spectrum`、`POST /api/samples/{id}/seal`
- 证据：`POST /api/evidence`、`GET /api/evidence`、`PUT /api/evidence/{id}`、
  `POST /api/evidence/{id}/validate`、`POST /api/evidence/{id}/exclude`
- 相似度：`POST /api/compare`、`GET /api/similarity`、`POST /api/similarity/recompute`
- 谱系：`POST /api/lineages`、`GET /api/lineages`、`GET /api/lineages/{id}`、
  `POST /api/lineages/{id}/edges`、`POST /api/lineage-edges/{id}/split`、
  `POST /api/lineages/{id}/merge`、`POST /api/lineages/{id}/confirm`、
  `POST /api/lineages/{id}/reject`、`POST /api/lineages/{id}/mutual-exclusive`、
  `POST /api/lineages/{id}/counterexamples`、`GET /api/lineages/{id}/counterexamples`
- 版本：`POST /api/versions`、`GET /api/versions`、`GET /api/versions/{id}`、
  `POST /api/versions/{id}/freeze`、`POST /api/versions/{id}/share`、
  `POST /api/versions/{id}/supersede`、`GET /api/versions/{id}/diff`
- 运维：`GET /api/health`、`GET /api/self-check`
