package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/akane9506/cinna/internal/app"
	"github.com/akane9506/cinna/internal/app/agent/core"
	"github.com/akane9506/cinna/internal/app/agent/memory"
	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/cloudwego/eino/compose"
)

type CinnaReactAgent struct {
	core   *core.AgentCore
	runner compose.Runnable[*core.GraphInput, *core.GraphOutput]
	store  *memory.MemoryStore
	logger *slog.Logger
}

// Create a new agent with graph and runner built
func NewCinnaReactAgent(
	ctx context.Context,
	config *app.Config,
	repos *ports.Repositories,
	logger *slog.Logger) (*CinnaReactAgent, error) {
	agent := new(CinnaReactAgent)
	core, err := core.InitializeAgentCore(ctx, config, repos, logger)
	if err != nil {
		return nil, err
	}
	agent.logger = logger
	agent.core = core
	agent.store = memory.NewMemoryStore(repos.AgentMemory, logger)
	// compile graph and create a runner
	runner, err := core.BuildGraph(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cinna agent runner: %w", err)
	}
	agent.runner = runner
	return agent, nil
}
