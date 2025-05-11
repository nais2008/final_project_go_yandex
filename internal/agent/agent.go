package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"github.com/nais2008/final_project_go_yandex/proto"
	"github.com/nais2008/project_go_yandex2/internal/config"
	"github.com/nais2008/project_go_yandex2/internal/models"
)

type Agent struct {
	cfg       config.Config
	grpcClient proto.OrchestratorServiceClient
	conn       *grpc.ClientConn
}

// NewAgent ...
func NewAgent(cfg config.Config) (*Agent, error) {
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}
	grpcClient := proto.NewOrchestratorServiceClient(conn)

	return &Agent{
		cfg:        cfg,
		grpcClient: grpcClient,
		conn:       conn,
	}, nil
}

// Run ...
func (a *Agent) Run() {
	for i := 0; i < a.cfg.ComputingPower; i++ {
		go a.processTasks()
	}
}

// processTasks ...
func (a *Agent) processTasks() {
	for {
		task, err := a.getTask()
		if err != nil {
			log.Printf("Ошибка получения задачи: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if task.ID == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		result := a.ComputeTask(task)
		time.Sleep(time.Duration(a.getOperationTime(task.Operation)) * time.Millisecond)

		a.submitResult(task.ID, result)
	}
}

// getTask ...
func (a *Agent) getTask() (models.Task, error) {
	req := &proto.TaskRequest{
		UserId: 1,
	}
	resp, err := a.grpcClient.GetTask(context.Background(), req)
	if err != nil {
		return models.Task{}, fmt.Errorf("failed to get task: %v", err)
	}

	return models.Task{
		ID:       resp.GetTaskId(),
		Arg1:     resp.GetArg1(),
		Arg2:     &resp.GetArg2(),
		Operation: resp.GetOperation(),
	}, nil
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

// submitResult ...
func (a *Agent) submitResult(taskID int, result float64) {
	req := &proto.TaskResultRequest{
		TaskId: taskID,
		Result: result,
	}
	_, err := a.grpcClient.SubmitTaskResult(context.Background(), req)
	if err != nil {
		log.Printf("Ошибка отправки результата: %v", err)
	}
}

// getOperationTime ...
func (a *Agent) getOperationTime(operation string) int {
	switch operation {
	case "+":
		return a.cfg.TimeAdditionMS
	case "-":
		return a.cfg.TimeSubtractionMS
	case "*":
		return a.cfg.TimeMultiplicationMS
	case "/":
		return a.cfg.TimeDivisionMS
	default:
		return 0
	}
}

// Close ...
func (a *Agent) Close() {
	a.conn.Close()
}
