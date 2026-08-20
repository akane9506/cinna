<div align="center">

<svg width="180" height="180" style="border-radius: 50%;">
    <img src="./assets/cinna-small.jpg" width="180px" height="180px"/>
</svg>

<h1>Cinna</h1>

![CI](https://github.com/akane9506/cinna/actions/workflows/ci.yml/badge.svg)
![Deploy](https://github.com/akane9506/cinna/actions/workflows/deployment.yml/badge.svg)

[English](./README.md) | 简体中文

</div>

Cinna 是一个基于 Go 和 Eino 构建、可自定义的 Telegram ReAct 智能体机器人。它支持可配置的子智能体和模块以满足多种使用场景；通过数据库支持的记忆管理日常任务，同时在对话间保留长期上下文；并使用加密保护历史消息，以提升隐私和安全性。

## 智能体流程图

[Mermaid 流程图](https://mermaid.ai/d/a442ee33-6405-4268-84cb-9b0fb390b6fe)

```mermaid
flowchart LR
    Start(["START"]) --> InputProcessLambda["`**Input Process Lambda**<br><br>*Process input and store necessary info in the state*<br><br> In: *GraphInput<br>Out: []*schema.Message`"]
    InputProcessLambda --> IntentClassifier["`**Intent Classification**<br><br>*classifies user intention and possible action*<br><br>In: []*schema.Message<br>Out: *schema.Message*`"]
    IntentClassifier --> IntentLambda["`**Intent Validation Lambda**<br><br>*Validates intent classificaion node output, and feed intention to the output branching*<br><br>In: *schema.Message<br>Out: []*schema.Message`"]
    IntentLambda --> IntentRouter["`**Branch by Intent**<br>Intent:<br>SHOPPING | FEEDBACK | OTHER<br>`"]
    IntentRouter -- SHOPPING --> ListLambda["<b>List Lambda</b><br><br><i>List contents in the shopping list table <br>(note: 1 - list expired items, 2: items are with categories, notify cinna to responde with the corresponding category)</i><br><br>In: []*schema.Message<br>Out: []*schema.Message"]
    ListLambda --> ActionRouter["`**Branch by Action Type**<br>Action:<br>LIST | UPDATE | None<br>`"]
    ActionRouter -->|UPDATE| ShoppingPlanner["`**Shopping Planner**<br>(we don't save this output into history state)<br><br>*Reads user-cinna chat history and decide the Shopping DB operation*<br><br>In: []*schema.Message<br>Out: *schema.Message`"]
    ActionRouter -- LIST --> Passthrough(["Passthrough Node"])
    ShoppingPlanner --> UpdateShoppingListLambda["`**Update Shopping List Lambda**<br><br>Parse the output from the planner into an array of commands, then execute commands to update the DB shopping list<br><br>In:*schema.Message<br>out: []*schema.Message`"]
    IntentRouter -- FEEDBACK --> FeedbackListLambda["`**List Feedback Lambda**<br><br>Prepare pending/in-progress tasks for the Feedback Planner<br><br>In: []*schema.Message<br>Out: []*schema.Message`"]
    FeedbackActionRouter["<b>Branch by Action Type</b><br>Action:<br>List | UPDATE | NONE"] -->|"NONE | LIST (we don't support LIST command for feedbacks)"| Passthrough
    FeedbackListLambda --> FeedbackActionRouter
    FeedbackActionRouter -->|UPDATE| FeedbackPlanner["`**Feedback Planner**<br>(we don't save this output into history state as well)<br><br>Check the current user feedback lists, and decide which items should be added/updated.<br><br>In: []*schema.Message<br>Out: *schema.Message`"]
    FeedbackPlanner --> UpdateFeedbackLambda["<b>Update Feedback Items Lambda</b><br><br>Parse the output from the planner, then update the DB feedback list<br><br>In: *schema.Message<br>out: []*schema.Message"]
    UpdateShoppingListLambda --> Passthrough
    UpdateFeedbackLambda --> Passthrough
    IntentRouter -- OTHER --> Passthrough
    Passthrough --> MemoryCompressionRouter["<b>Conversation size check</b>"]
    MemoryCompressionRouter -- TRUE --> StartOfReplyAndCompressionChain["`**START**<br>Compression+Reply Chain`"]
    StartOfReplyAndCompressionChain --> MemoryCompressor["`**Memory Compressor**<br><br>Compresses chat history into enhanced notes.<br><br>In: []*schema.Message<br>Out: *schema.Message`"] & ReplyGenerator["`****Reply Generator****<br><br>Generates a reply based on the current turn.<br><br>In: []*schema.Message<br>Out: *schema.Message`"]
    ReplyGenerator -->|"key: cinna_reply"| CompressionChainPostProcessor["`**Chain Post Processor**<br><br>Waits for and combines the results from parallel nodes.<br><br>In: map[string]any<br>Out: *ReplyCompressionOutput`"]
    MemoryCompressor -->|"key: memory_compression"| CompressionChainPostProcessor
    MemoryCompressionRouter -- FALSE --> StartOfReplyOnlyChain["`**START**<br>Reply Only Chain`"]
    StartOfReplyOnlyChain --> ReplyGenerator & PassThroughNode(["Passthrough node"])
    PassThroughNode --> CompressionChainPostProcessor
    CompressionChainPostProcessor --> EndOfReplyAndCompressionChain["END\nCompression+Reply Chain"]
    EndOfReplyAndCompressionChain --> ProcessOutputLambda["<b>Process Output Lambda</b><br><br>Organizes and outputs chat history, response, and other metadata<br><br>In: *ReplyCompressionOutput<br>Out: *GraphOutput"]
    ProcessOutputLambda --> End(["END"])
```

## 功能

- 开发环境使用 Telegram 长轮询，生产环境使用 Webhook
- 收集反馈，并向规划器提供待处理和进行中反馈的上下文
- 提供帮助、允许名单管理及每日通知订阅开关命令
- 基于每位用户当前上下文生成个性化每日通知

## 对话窗口命令

- `/help`：显示帮助信息。
- `/notify on`：开启每日定时通知。
- `/notify off`：关闭每日定时通知。
- `/addmember <telegram_user_id>`：将用户加入允许名单（仅管理员）。目标用户须已启动过 Cinna Bot。

## 项目结构

```text
.
├── cmd/cinna/                 # 应用程序入口
├── db/
│   ├── schema.sql             # PostgreSQL 数据库结构
│   └── queries/               # sqlc 查询定义
├── internal/
│   ├── app/
│   │   ├── agent/             # 智能体图、节点和提示词
│   │   ├── ports/             # 仓储与应用层接口
│   │   ├── telegram/          # Telegram 更新、命令和通知处理
│   │   └── config.go          # 应用配置加载与校验
│   ├── postgres/              # PostgreSQL 仓储实现
│   ├── security/              # 消息加密相关功能
│   ├── server/                # Webhook 和事件监听 HTTP 服务
│   └── utils/                 # 通用工具与测试辅助代码
├── assets/                    # 项目图片资源
├── compose.yaml               # 本地 PostgreSQL 与镜像构建配置
├── Dockerfile                 # 容器镜像构建定义
├── sqlc.yaml                  # sqlc 配置
└── README.md                  # 英文文档
```

## 环境要求

- Go 1.26+
- Docker
- `sqlc`
- Telegram Bot Token
- A DeepSeek API key (用于生成回复)
- An OpenAI API key (用于语音识别以及JSON输出)

## 配置

创建并配置本地环境文件：

```bash
cp .env.example .env
```

加载前请编辑 `.env`。本地开发环境至少需要设置：

```dotenv
GO_ENV=development
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
APP_TIMEZONE=America/Los_Angeles
DATABASE_URL=postgres://cinna:change_me@localhost:5432/cinna?sslmode=disable
DEEPSEEK_API_KEY=your_deepseek_api_key
OPENAI_API_KEY=your_openai_api_key
MESSAGE_ENCRYPTION_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
MESSAGE_ENCRYPTION_KEY_ID=local-v1
POSTGRES_USER=cinna
POSTGRES_PASSWORD=change_me
POSTGRES_DB=cinna
WEBHOOK_SECRET=your_webhook_secret
```

示例加密密钥仅适用于本地开发。生产环境请使用 `openssl rand -base64 32` 生成唯一的 AES-256 密钥。部署时须持续保留该密钥及其稳定的 `MESSAGE_ENCRYPTION_KEY_ID`，以确保已有消息可被解密。

生产环境的 Webhook 模式必须同时设置 `WEBHOOK_URL` 和 `WEBHOOK_SECRET`：

```dotenv
GO_ENV=production
WEBHOOK_URL=https://your-public-host.example
WEBHOOK_SECRET=your_webhook_secret
# 可选，默认值为 8080。
PORT=8080
```

加载环境变量：

```bash
set -a
source .env
set +a
```

## 数据库设置

启动 PostgreSQL：

```bash
docker compose up -d postgres
```

首次启动时，PostgreSQL 会创建由 `POSTGRES_DB` 指定的数据库。通过命令行验证服务和数据库：

```bash
docker compose exec postgres \
  pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"

docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT current_database(), current_user;"
```

应用数据库结构：

```bash
docker compose exec -T postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" < db/schema.sql
```

验证所需表和购物类别类型是否存在：

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "\d allowed_users" \
  -c "\d shopping_list" \
  -c "\d agent_memory" \
  -c "\d feedbacks" \
  -c "\dT+ shopping_category"
```

根据 SQL 查询生成 Go 代码：

```bash
sqlc generate
```

## Telegram 允许名单

Cinna 会使用 `allowed_users` 表检查每一条 Telegram 更新。用户与机器人交互前，请先将用户添加到允许名单，或重新激活该用户：

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "INSERT INTO allowed_users (telegram_user_id, role)
      VALUES (12345678, 'admin')
      ON CONFLICT (telegram_user_id) DO UPDATE
      SET role = EXCLUDED.role, is_active = TRUE, updated_at = NOW();"
```

列出允许的用户：

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT * FROM allowed_users ORDER BY updated_at;"
```

购物清单的记录关联至 `allowed_users`，因此 Cinna 只能为允许名单中的 Telegram 用户存储条目。

## 运行

使用 Telegram 长轮询在本地运行 Cinna：

```bash
go run ./cmd/cinna
```

生产模式下，Cinna 会配置 Telegram Webhook、启动 Webhook 接收器，并通过 `PORT` 或默认端口 `8080` 提供服务。

## 每日定时通知

Cinna 提供 `POST /internal/daily-notification` 端点，向所有通过 `/notify on`
开启通知的用户发送每日提醒。请求必须包含 `X-Cinna-Webhook-Secret` 请求头，
其值须与 `WEBHOOK_SECRET` 一致。

### 本地开发

在本地启动 Cinna 后，可使用以下命令手动触发通知：

```bash
curl --request POST http://localhost:8080/internal/daily-notification \
  --header "X-Cinna-Webhook-Secret: $WEBHOOK_SECRET"
```

### 使用 Google Cloud Scheduler 的生产环境

创建一个 Cloud Scheduler HTTP 任务，调用已部署服务的公开 URL。请根据期望的
通知发送时间配置执行计划和时区：

```bash
gcloud scheduler jobs create http cinna-daily-notification \
  --location="$REGION" \
  --schedule="0 9 * * *" \
  --time-zone="America/Los_Angeles" \
  --uri="https://your-service-url/internal/daily-notification" \
  --http-method=POST \
  --headers="X-Cinna-Webhook-Secret=$WEBHOOK_SECRET"
```

上述示例会在 `America/Los_Angeles` 时区每天 09:00 执行。请按需替换时区、
执行计划和服务 URL。Scheduler 任务配置中包含请求头，因此请使用专用的
`WEBHOOK_SECRET`，并限制能够查看或管理该任务的用户权限。

## 容器镜像

多阶段 `Dockerfile` 会构建静态链接的 Linux `amd64` 二进制文件，并将其复制到精简的 Alpine 运行时镜像中。可在本地构建：

```bash
docker build -t cinna:latest .
```

使用 Compose 暴露的主机 PostgreSQL 端口，以开发模式运行镜像。在 Docker Desktop 中，`host.docker.internal` 会解析为宿主机：

```bash
docker compose up -d postgres

docker run --rm \
  --env-file .env \
  -e DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@host.docker.internal:5432/${POSTGRES_DB}?sslmode=disable" \
  cinna:latest
```

`gcp-deployment` Compose 服务会为 Google Artifact Registry 构建并标记镜像。构建和推送前，请设置仓库位置、Google Cloud 项目以及可选的不可变镜像标签：

```bash
export REGION=us-central1
export PROJECT_ID=your-google-cloud-project
export IMAGE_TAG=$(git rev-parse --short HEAD)

gcloud auth configure-docker "${REGION}-docker.pkg.dev"
gcloud artifacts repositories describe cinna \
  --location "$REGION" \
  --project "$PROJECT_ID"

docker compose build gcp-deployment
docker compose push gcp-deployment
```

名为 `cinna` 的 Artifact Registry 仓库必须预先创建。此 Compose 服务负责构建和发布镜像；将镜像部署到 Cloud Run 等运行环境是另一个步骤。生产运行时必须提供[配置](#配置)中列出的变量，包括 HTTPS Webhook URL、Webhook 密钥，以及部署容器可以访问的数据库 URL。

## SQL 命令

打开 PostgreSQL Shell：

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
```

修改 `db/schema.sql` 或 `db/queries/` 下的文件后，重新生成代码：

```bash
sqlc generate
```

停止 PostgreSQL：

```bash
docker compose down
```

删除 PostgreSQL 数据：

```bash
docker compose down -v
```

## 检查

```bash
go test ./...
go vet ./...
```

## 手动 LLM 测试

手动模型和智能体测试受 `internal/utils.EnforceManualTest` 保护。只有设置 `RUN_MANUAL_TEST=1` 时才会执行，并且需要提供 `DEEPSEEK_API_KEY` 以及 `OPENAI_API_KEY`。
