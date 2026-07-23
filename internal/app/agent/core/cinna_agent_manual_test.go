package core

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log/slog"
// 	"os"
// 	"testing"

// 	"github.com/akane9506/cinna/internal/utils"
// 	"github.com/cloudwego/eino/schema"
// )

// const mockTelegramUserID = 181496207

// func TestIntentLambda(t *testing.T) {
// 	ctx := context.Background()
// 	agent, err := setupAgent(t, ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	runnable, err := buildIntentLambdaOutputNode(ctx, agent, t)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	input := &TaskInput{
// 		telegramUserID: mockTelegramUserID,
// 	}
// 	t.Log("======== Shopping intention lambda testing 1 ========")
// 	input.chatHistory = []*schema.Message{
// 		&schema.Message{Role: schema.Assistant, Content: "做蛋糕需要用到面粉和糖"},
// 		&schema.Message{Role: schema.User, Content: "可以帮我把这些记下吗？"},
// 	}
// 	_, err = runnable.Invoke(ctx, input)
// 	t.Log("======== Shopping intention lambda testing 2 ========")
// 	input.chatHistory = []*schema.Message{
// 		&schema.Message{Role: schema.User, Content: "可以看一下我都要买什么吗？"},
// 	}
// 	_, err = runnable.Invoke(ctx, input)
// 	t.Log("======== Other intention lambda testing ========")
// 	input.chatHistory = []*schema.Message{
// 		&schema.Message{Role: schema.User, Content: ""},
// 	}
// 	_, err = runnable.Invoke(ctx, input)
// 	t.Log("======== Other intention lambda testing ========")
// 	input.chatHistory = []*schema.Message{
// 		&schema.Message{Role: schema.User, Content: "拍摄模型车需要用到什么设备呀？"},
// 	}
// 	_, err = runnable.Invoke(ctx, input)
// 	t.Log("======== Feedback intention lambda testing ========")
// 	input.chatHistory = []*schema.Message{
// 		&schema.Message{Role: schema.User, Content: "Cinna目前的回复速度好慢呀？"},
// 	}
// 	_, err = runnable.Invoke(ctx, input)
// }

// func TestShoppingTaskPlanningNode(t *testing.T) {
// 	ctx := context.Background()
// 	agent, err := setupAgent(t, ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	runner, err := buildShoppingTaskPlanerGraph(ctx, agent)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	input := &TaskInput{telegramUserID: mockTelegramUserID}
// 	t.Log("========== Test Adding Items ==========")
// 	input.chatHistory = []*schema.Message{
// 		{Role: schema.User, Content: "Cinna可以帮我把牛奶和低筋面粉,和NyQuil加到购物车吗"},
// 	}
// 	result, err := runner.Invoke(ctx, input)
// 	t.Log(result.outputMessage.Content)
// 	t.Log("========== DB manipulation test ==========")
// 	msgs := []*schema.Message{
// 		{
// 			Role:    schema.User,
// 			Content: "帮我把牛奶删掉，再加一盒彩色马克笔，然后把 notebook 改成 A4 notebook",
// 		},
// 		{
// 			Role:    schema.Assistant,
// 			Content: "items in the current database: {activeItems: [{id: 1, category: grocery, name: milk}, {id: 2, category: pharmacy, name: bandages}, {id: 3, category: pet_store, name: cat food}], expiredItems: [{id: 4, category: toy_shop, name: lego set}, {id: 5, category: stationery, name: notebook}]}",
// 		},
// 	}
// 	/*
// 		The result should look like:
// 		  "commands": [
// 		    {
// 		      "id": "1",
// 		      "method": "REMOVE",
// 		      "category": "grocery",
// 		      "name": "milk"
// 		    },
// 		    {
// 		      "id": "",
// 		      "method": "ADD",
// 		      "category": "stationery",
// 		      "name": "一盒彩色马克笔(marker)"
// 		    },
// 		    {
// 		      "id": "5",
// 		      "method": "MODIFY",
// 		      "category": "stationery",
// 		      "name": "A4 notebook"
// 		    }
// 		  ]*/
// 	input.chatHistory = msgs
// 	result, err = runner.Invoke(ctx, input)
// 	t.Log(result.outputMessage.Content)
// }

// func TestFeedbackPlannerNode(t *testing.T) {
// 	ctx := context.Background()
// 	agent, err := setupAgent(t, ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	runnable, err := buildFeedbacksPlannerNode(ctx, agent, t)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	input := &TaskInput{telegramUserID: mockTelegramUserID}
// 	// ========== Test Single Feedback ==========
// 	t.Log("\n" + strings.Repeat("=", 20) + "Test Single Feedback" + strings.Repeat("=", 20))
// 	input.chatHistory = []*schema.Message{
// 		{Role: schema.User, Content: "Cinna反应的速度有点慢呀"},
// 	}
// 	result, err := runnable.Invoke(ctx, input)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Log(result.outputMessage.Content)

// 	// ========== Test Single Feedback With Conversation ==========
// 	t.Log("\n" + strings.Repeat("=", 20) + "Test Single Feedback With Conversation" + strings.Repeat("=", 20))
// 	input.chatHistory = []*schema.Message{
// 		{Role: schema.User, Content: "Cinna我想吃巧克力"},
// 		{Role: schema.User, Content: "好的已经把巧克力加进小本本啦"},
// 		{Role: schema.User, Content: "不对你应该和我确认一下我要那种巧克力！"},
// 	}
// 	result, err = runnable.Invoke(ctx, input)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Log(result.outputMessage.Content)

// 	// ========== Test Multiple Feedbacks With Conversation ==========
// 	t.Log("\n" + strings.Repeat("=", 20) + "Test Single Feedback With Conversation" + strings.Repeat("=", 20))
// 	input.chatHistory = []*schema.Message{
// 		{Role: schema.User, Content: "Cinna我想吃雪糕"},
// 		{Role: schema.User, Content: "好的已经把雪糕加进小本本啦"},
// 		{Role: schema.User, Content: "不对你应该和我确认一下我要哪种雪糕，还有你的表情太多啦！"},
// 	}
// 	result, err = runnable.Invoke(ctx, input)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	t.Log(result.outputMessage.Content)
// }

// // ========== Graph-Building For Manual Tests [DO NOT USE IN PRODUCTION] ==========
// func buildCinnaChatGraph(
// 	ctx context.Context, agent CinnaReactAgent) (compose.Runnable[*TaskInput, *TaskOutput], error) {
// 	if agent.runner != nil {
// 		return nil, fmt.Errorf("this is only for manual test, do not use it in production")
// 	}
// 	graph := agent.graph
// 	agent.AddProcessInputLambdaNode()
// 	agent.AddCinnaResponseNode()
// 	agent.AddProcessOutputLambdaNode()
// 	graph.AddEdge(compose.START, processInputLambdaNodeName)
// 	graph.AddEdge(processInputLambdaNodeName, cinnaChatNodeName)
// 	graph.AddEdge(cinnaChatNodeName, processOutputLambdaNodeName)
// 	graph.AddEdge(processOutputLambdaNodeName, compose.END)
// 	runnable, err := graph.Compile(ctx)
// 	if err != nil {
// 		agent.logger.Error("failed to compile manual cinna chat graph", "error", err)
// 		return nil, err
// 	}
// 	return runnable, nil
// }

// func buildShoppingTaskPlanerGraph(
// 	ctx context.Context, agent *CinnaReactAgent) (compose.Runnable[*TaskInput, *TaskOutput], error) {
// 	if agent.runner != nil {
// 		return nil, fmt.Errorf("this is only for manual test, do not use it in production")
// 	}
// 	graph := agent.graph
// 	agent.AddProcessInputLambdaNode()
// 	agent.AddShoppingTaskPlanningNode()
// 	agent.AddProcessOutputLambdaNode()
// 	graph.AddEdge(compose.START, processInputLambdaNodeName)
// 	graph.AddEdge(processInputLambdaNodeName, shoppingTasksPlannerLLMNodeName)
// 	graph.AddEdge(shoppingTasksPlannerLLMNodeName, processOutputLambdaNodeName)
// 	graph.AddEdge(processOutputLambdaNodeName, compose.END)
// 	runnable, err := graph.Compile(ctx)
// 	if err != nil {
// 		agent.logger.Error("failed to compile manual cinna chat graph", "error", err)
// 		return nil, err
// 	}
// 	return runnable, nil
// }

// func buildIntentLambdaOutputNode(
// 	ctx context.Context, agent *CinnaReactAgent, t *testing.T) (compose.Runnable[*TaskInput, *TaskOutput], error) {
// 	if agent.runner != nil {
// 		return nil, fmt.Errorf("this is only for manual test, do not use it in production")
// 	}
// 	// add a test-purposed node to make the test graph type compatible with the
// 	// agent's I/O type
// 	graph := agent.graph
// 	agent.AddProcessInputLambdaNode()
// 	agent.AddIntentClassificationNode()
// 	agent.AddIntentOutputLambdaNode()
// 	graph.AddLambdaNode("mock_final_output",
// 		compose.InvokableLambda[[]*schema.Message, *schema.Message](func(
// 			ctx context.Context, input []*schema.Message) (*schema.Message, error) {
// 			compose.ProcessState[*CinnaAgentState](ctx, func(
// 				ctx context.Context,
// 				state *CinnaAgentState,
// 			) error {
// 				t.Log("Intent: ", state.ChatIntent)
// 				t.Log("Action: ", state.ActionType)
// 				t.Log("Messages: ", state.History)
// 				return nil
// 			})
// 			return input[len(input)-1], nil
// 		}),
// 	)
// 	agent.AddProcessOutputLambdaNode()
// 	graph.AddEdge(compose.START, processInputLambdaNodeName)
// 	graph.AddEdge(processInputLambdaNodeName, intentLLMNodeName)
// 	graph.AddEdge(intentLLMNodeName, intentLambdaNodeName)
// 	graph.AddEdge(intentLambdaNodeName, "mock_final_output")
// 	graph.AddEdge("mock_final_output", processOutputLambdaNodeName)
// 	graph.AddEdge(processOutputLambdaNodeName, compose.END)
// 	runnable, err := graph.Compile(ctx)
// 	if err != nil {
// 		t.Error("failed to compile manual intent lambda graph", "error", err)
// 		return nil, err
// 	}
// 	return runnable, nil
// }

// func buildFeedbacksPlannerNode(
// 	ctx context.Context,
// 	agent *CinnaReactAgent,
// 	t *testing.T) (compose.Runnable[*TaskInput, *TaskOutput], error) {
// 	if agent.runner != nil {
// 		return nil, fmt.Errorf("this is only for manual test, do not use it in production")
// 	}
// 	agent.AddProcessInputLambdaNode()
// 	agent.AddFeedbacksPlanningNode()
// 	agent.AddProcessOutputLambdaNode()

// 	graph := agent.graph
// 	graph.AddEdge(compose.START, processInputLambdaNodeName)
// 	graph.AddEdge(processInputLambdaNodeName, feedbacksUpdatePlannerLLMNodeName)
// 	graph.AddEdge(feedbacksUpdatePlannerLLMNodeName, processOutputLambdaNodeName)
// 	graph.AddEdge(processOutputLambdaNodeName, compose.END)
// 	runnable, err := graph.Compile(ctx)
// 	if err != nil {
// 		t.Error("failed to compile manual feedback planner graph", "error", err)
// 	}
// 	return runnable, nil
// }

// ========== Agent / Model Setup ==========

// func setupDeepseekModelTesting(t *testing.T) (*Models, context.Context) {
// 	utils.EnforceManualTest(t)
// 	apiKey := os.Getenv("DEEPSEEK_API_KEY")
// 	if apiKey == "" {
// 		t.Fatal("DEEPSEEK_API_KEY s required")
// 	}
// 	ctx := context.Background()
// 	models := NewLLMModels(&APIKey{
// 		Deepseek: apiKey,
// 	}, slog.Default())
// 	return models, ctx
// }

// func setupAgent(t *testing.T, ctx context.Context) (*AgentState, error) {
// 	utils.EnforceManualTest(t)
// 	config, err := app.LoadConfig()
// 	if err != nil {
// 		return nil, err
// 	}
// 	agent, err := initializeBaseAgent(ctx, config, slog.New(slog.NewTextHandler(io.Discard, nil)))
// 	if err != nil {
// 		return nil, err
// 	}
// 	return agent, nil
// }
