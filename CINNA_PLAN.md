# Cinna Plan

## Architecture

```text
Telegram
  -> Application
  -> Eino Graph
       -> Chat Model
       -> Tool Validation
       -> Tool Execution
       -> Chat Model
  -> Telegram Response
```

Use Eino `compose.Graph` to implement a bounded ReAct workflow. Do not use ADK,
prebuilt agents, or multiple agents.

## Project Structure

```text
cinna/
├── cmd/cinna/main.go
├── internal/
│   ├── app/                  # configuration and dependency wiring
│   ├── agent/                # Eino graph, state, and tool routing
│   ├── shopping/             # domain services and repository interfaces
│   └── platform/
│       ├── postgres/         # PostgreSQL repositories
│       └── telegram/         # Telegram adapter
├── db/
│   ├── migrations/
│   └── queries/
├── prompts/
├── deployments/
├── go.mod
└── README.md
```

Create directories only when they are needed.

## Rules

- Keep Telegram types inside the Telegram adapter.
- Keep Eino types inside the agent package.
- Tools call domain services, never SQL directly.
- Validate and authorize every tool call.
- Derive user identity from trusted Telegram context.
- Generate replies from committed operation results.
- Bound graph iterations, execution time, and conversation context.

## Milestones

### 1. Bootstrap

- Initialize the Go service.
- Add configuration, logging, shutdown, and tests.

### 2. Telegram

- Connect using long polling.
- Receive and reply to text messages.
- Restrict access to allowed users.

### 3. Agent Graph

Build this graph:

```text
START
  -> prepare_messages
  -> chat_model
       -> END             if the model returns a final response
       -> validate_tools  if the model requests tools
  -> execute_tools
  -> chat_model
```

- Add the persona prompt.
- Store messages in graph state.
- Limit iterations and execution time.
- Return a safe fallback when execution fails.

### 4. PostgreSQL

- Add PostgreSQL, migrations, and typed queries.
- Implement shopping domain services.
- Test services independently from the graph.

### 5. Shopping Tools

Start with:

```text
add_shopping_items
list_shopping_items
```

- Validate tool arguments.
- Use the authenticated Telegram user.
- Call shopping services.
- Return committed operation results.

### 6. Reliability

- Deduplicate Telegram updates.
- Add timeouts, error handling, and observability.
- Require confirmation for destructive operations.

### 7. Extend

Add features only after the first workflow works end to end:

- Complete and remove items.
- Clear completed items.
- Conversation history.
- Feedback.
- Voice input.
- Production deployment.

## First Target

```text
Telegram message
  -> Eino Graph
  -> add/list shopping tool
  -> PostgreSQL
  -> Telegram response
```

Defer ADK, prebuilt agents, multiple agents, Redis, queues, and vector databases.
