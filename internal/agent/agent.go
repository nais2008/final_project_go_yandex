package agent

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	proto "github.com/nais2008/final_project_go_yandex/internal/protos/gen/go/sso"
)

type Agent struct {
    cfg    config.Config
    client proto.AgentClient
}

func NewAgent(cfg config.Config) *Agent {
    const op = "agent.NewAgent"

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    conn, err := grpc.DialContext(ctx, cfg.GrpcServerAddr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithBlock(),
    )
    if err != nil {
        slog.Error(op, "failed to connect to gRPC server", err)
        return nil
    }

    client := proto.NewAgentClient(conn)

    slog.Info(op, "agent created")

    return &Agent{cfg: cfg, client: client}
}

func (a *Agent) Run() {
    const op = "agent.Run"

    for {
        task, err := a.getTask()
        if err != nil || task == nil {
            slog.Error(op, "failed to get task", err)
            time.Sleep(1 * time.Second)
            continue
        }

        result := a.ComputeTask(task)
        time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)

        a.submitResult(task.Id, result)
    }
}

func (a *Agent) getTask() (*proto.Task, error) {
    const op = "agent.getTask"

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    resp, err := a.client.GetTask(ctx, &proto.GetTaskRequest{})
    if err != nil {
        slog.Error(op, "failed to get task from server", err)
        return nil, err
    }

    slog.Info(op, "task received")

    return resp.Task, nil
}

// ComputeTask ...
func (a *Agent) ComputeTask(task *proto.Task) float64 {
    const op = "agent.ComputeTask"

    if task.Arg2 == 0 {
        return task.Arg1
    }

    var result float64

    switch task.Operation {
    case "+":
        result = task.Arg1 + task.Arg2
    case "-":
        result = task.Arg1 - task.Arg2
    case "*":
        result = task.Arg1 * task.Arg2
    case "/":
        if task.Arg2 == 0 {
            return 0
        }
        result = task.Arg1 / task.Arg2
    default:
        slog.Warn(op, "unknown operation", "operation", task.Operation)
        return 0
    }

    slog.Info(op, "task computed", "result", result)
    return result
}

func (a *Agent) submitResult(taskID int32, result float64) {
    const op = "agent.submitResult"

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    _, err := a.client.SubmitResult(ctx, &proto.SubmitResultRequest{
        Id:     taskID,
        Result: result,
    })

    if err != nil {
        slog.Error(op, "failed to submit result", err)
    } else {
        slog.Info(op, "result submitted", "taskID", taskID)
    }
}
