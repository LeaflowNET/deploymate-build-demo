# deploymate-build-demo

用于测试 DeployMate 源码自动构建的独立 Go HTTP 服务，无数据库和第三方依赖。

## DeployMate 构建配置

- 分支：`main`
- 构建上下文：`.`
- Dockerfile：`Dockerfile`
- 容器端口：`8080`
- 健康检查：`GET /healthz`（200，正文 `ok`）
- 可选环境变量：`DEMO_MESSAGE`（首页返回的消息）、`PORT`（默认 `8080`）
- 可选构建参数：`VERSION`、`COMMIT`（首页返回构建版本，默认 `dev` / `unknown`）

访问 `/` 返回 JSON：服务名、消息、构建版本、提交标识、容器主机名和启动时间。
`VERSION` / `COMMIT` 需要构建方显式传入，不会自动读取 Git 元数据。

## 本地运行

```sh
go run .
curl http://localhost:8080/
curl http://localhost:8080/healthz
```

## Docker Compose

```sh
docker compose up -d --build
curl http://localhost:18080/
docker compose down
```

默认映射主机 `18080` 到容器 `8080`；可用 `DEMO_PORT` 修改主机端口。

## 验证自动更新

修改 `main.go` 中默认的 `Hello from DeployMate!`，提交并推送，再触发自动构建。
部署完成后访问 `/`，检查消息变化及新的 `startedAt`。
也可以在部署配置中设置 `DEMO_MESSAGE` 验证环境变量注入；设置后它会覆盖代码中的默认消息。
