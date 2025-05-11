package agent

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/models"
	proto "github.com/nais2008/final_project_go_yandex/internal/protos/gen/go/sso"
	"os"
)

// Agent ...
type Agent struct {
	cfg    config.Config
	client proto.AgentServiceClient
	conn   *grpc.ClientConn
	token  string
}

// NewAgent ...
func NewAgent(cfg config.Config) *Agent {
	conn, err := grpc.Dial(cfg.GrpcServerAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("could not connect to gRPC server: %v", err)
	}
	client := proto.NewAgentServiceClient(conn)

	token := os.Getenv("JWT_TOKEN")
	if token == "" {
		log.Fatalf("JWT_TOKEN must be set in environment")
	}

	ag := &Agent{
		cfg:    cfg,
		client: client,
		conn:   conn,
		token:  token,
	}

	return ag
}

// Run ...
func (a *Agent) Run() error {
	defer a.conn.Close()
	for {
		task, err := a.getTask()
		if err != nil {
			log.Printf("error getting task: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if task.ID == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		result := a.ComputeTask(task)
		a.submitResult(int(task.ID), result)

		time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)
	}
}

func (a *Agent) getTask() (models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+a.token)

	resp, err := a.client.GetTask(ctx, &proto.TaskRequest{})
	if err != nil {
		return models.Task{}, err
	}

	return models.Task{
		ID:        uint(resp.TaskId),
		Arg1:      resp.Arg1,
		Arg2:      floatPointer(resp.Arg2),
		Operation: resp.Operation,
	}, nil
}

func floatPointer(f float64) *float64 {
	return &f
}

// ComputeTask ...
func (a *Agent) ComputeTask(task models.Task) float64 {
	if task.Arg2 == nil {
		return task.Arg1
	}
	switch task.Operation {
	case "+":
		return task.Arg1 + *task.Arg2
	case "-":
		return task.Arg1 - *task.Arg2
	case "*":
		return task.Arg1 * *task.Arg2
	case "/":
		if *task.Arg2 == 0 {
			return 0
		}
		return task.Arg1 / *task.Arg2
	default:
		return 0
	}
}

func (a *Agent) submitResult(taskID int, result float64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+a.token)

	req := &proto.TaskResultRequest{
		TaskId: int64(taskID),
		Result: result,
	}

	_, err := a.client.SubmitTaskResult(ctx, req)
	if err != nil {
		log.Printf("error submitting result for task %d: %v", taskID, err)
	}
}
