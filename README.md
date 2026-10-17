# Taskmate · DeployMate build demo

一个完整可操作的任务管理测试项目：Vue 3 + TypeScript + Vite 前端、Go HTTP API、文件持久化、多阶段 Docker 构建。用来验证源码拉取、前端依赖安装和编译、后端构建、健康检查、数据卷和更新部署。

## 功能

- 创建、编辑、删除任务，待办 / 进行中 / 已完成三种状态。
- 高 / 中 / 低优先级，关键词搜索、状态和优先级筛选，每页 20 项。
- 全局任务数量统计，空数据、提交中、失败重试反馈。
- 简体中文、繁体粤语和英语，浅色 / 深色主题，桌面和手机布局。
- 部署信息显示构建版本、提交标识、启动时间和自定义消息。
- 数据写入文件后才返回成功，并通过临时文件、同步和原子重命名保存；并发写入由互斥锁串行化。

这是**共享、无需登录的测试应用**，请只在可信测试网络使用，不保存敏感信息。每个数据目录只运行一个实例，最多 2000 项任务，不支持多个进程同时写同一个数据卷。

## DeployMate 自动构建配置

| 配置 | 值 |
| --- | --- |
| 分支 | `main` |
| 构建上下文 | `.` |
| Dockerfile | `Dockerfile` |
| 容器端口 | `8080` |
| HTTP 健康检查 | `GET /healthz`，200 + `ok` |
| 持久化挂载路径 | `/data`（必须保留同一个数据卷以便更新后恢复任务） |
| 可选环境变量 | `DEMO_MESSAGE`、`PORT`（8080）、`DATA_DIR`（镜像中为 `/data`） |
| 可选构建参数 | `VERSION`、`COMMIT`（默认 `dev` / `unknown`，由构建方显式传入） |

Dockerfile 先用 Node 安装并编译前端，再由 Go 嵌入静态资源生成单个二进制。最终镜像基于 scratch，以非 root 用户运行，内置健康检查。不需要另开前端容器，也不需要外部数据库。

## 快速启动

```sh
docker compose up -d --build
# 打开 http://127.0.0.1:18080
```

Compose 默认仅监听主机回环地址；部署到测试服务器时，可通过现有反向代理访问或按测试需要修改端口绑定。停止使用 `docker compose down`，数据卷会保留；`docker compose down -v` 会删除任务数据。

## 本地开发

需要 Go 1.26、Node.js 22.12+、npm。

```sh
make build
PORT=18081 ./bin/taskmate
# 打开 http://127.0.0.1:18081
```

默认数据目录为当前目录下的 `data/`。前端开发可以另开终端：

```sh
cd web
npm run dev
# http://127.0.0.1:15174，API 代理至 18081
```

## 检查

```sh
cd web && npm ci && cd ..
make check
# 前端 ESLint、交互状态测试、类型检查和构建、Go 格式检查、vet、race 测试、后端构建

docker build --build-arg VERSION=v1-test --build-arg COMMIT=local -t taskmate:test .
python3 scripts/smoke.py taskmate:test
# 验证接口、真实容器替换后的数据保留、健康检查；仅清理脚本创建的容器和卷
```

## API

- `GET /healthz`：存活检查。
- `GET /api/info`：部署版本、提交标识、启动时间、消息。
- `GET /api/tasks?q=&status=&priority=&offset=0&limit=20`：任务列表与统计。`limit` 为 1–100，默认 50。
- `POST /api/tasks`：创建任务。
- `PUT /api/tasks/{id}`：完整更新任务。
- `DELETE /api/tasks/{id}`：删除任务。

创建 / 更新请求示例：

```json
{"title":"验证自动构建","description":"重建容器后任务仍然存在","status":"todo","priority":"high"}
```

标题为 1–200 字，描述最多 2000 字。更新采用最后写入生效；页面刷新可读取其他浏览器的改动。

## 建议验收顺序

1. 连接此仓库并完成首次构建部署，打开首页创建任务。
2. 修改页面文案或默认 `DEMO_MESSAGE`，提交推送并触发更新部署。
3. 打开部署信息核对 `VERSION` / `COMMIT`（如果构建方设置了它们），确认新版本内容生效。
4. 确认旧任务保留，继续编辑、切换状态和删除。
5. 修改部署环境变量 `DEMO_MESSAGE` 并重新创建容器，展开部署信息确认环境变量生效。
6. 检查手机布局、英文和深色主题，必要时停止容器验证平台失败状态与重试。
