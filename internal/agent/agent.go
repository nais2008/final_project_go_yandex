package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/models"
	pb "github.com/nais2008/final_project_go_yandex/internal/protos/gen/go/sso" // <--- ВАЖНО: Проверьте путь
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Agent представляет собой gRPC-клиента агента.
type Agent struct {
	cfg                config.Config
	orchestratorClient pb.OrchestratorServiceClient
}

// NewAgent создает новый экземпляр gRPC-агента.
func NewAgent(cfg config.Config) *Agent {
	conn, err := grpc.Dial(cfg.GrpcServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Agent failed to connect to Orchestrator gRPC server at %s: %v", cfg.GrpcServerAddr, err)
	}
	client := pb.NewOrchestratorServiceClient(conn)
	return &Agent{
		cfg:                cfg,
		orchestratorClient: client,
	}
}

// Run запускает основной цикл работы gRPC-агента.
func (a *Agent) Run() {
	log.Printf("gRPC Agent started. Connecting to Orchestrator at: %s", a.cfg.GrpcServerAddr)
	pollInterval := time.Second * 5
	log.Printf("Task poll interval: %s", pollInterval)

	for {
		task, err := a.getTask()
		if err != nil {
			log.Printf("Error getting task: %v", err)
			time.Sleep(pollInterval)
			continue
		}

		if task != nil && task.ID != 0 {
			log.Printf("Received task: %+v", task)
			result, err := a.ComputeTask(task)
			if err != nil {
				log.Printf("Error computing task %d: %v", task.ID, err)
				a.submitResult(task.ID, 0)
			} else {
				a.submitResult(task.ID, result)
			}
			time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)
		} else {
			log.Println("No tasks available. Sleeping...")
			time.Sleep(pollInterval)
		}
	}
}

// getTask запрашивает новую задачу у оркестратора через gRPC.
func (a *Agent) getTask() (*models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	req := &pb.TaskRequest{
		UserId: 1, // Вам может потребоваться передавать фактический UserID
	}

	resp, err := a.orchestratorClient.GetTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error calling GetTask: %w", err)
	}

	if resp.TaskId == 0 {
		return nil, nil // Нет доступных задач
	}

	task := &models.Task{
		ID:            uint(resp.TaskId),
		Arg1:          resp.Arg1,
		Operation:     resp.Operation,
		OperationTime: int(resp.OperationTime),
	}
	if resp.Arg2 != 0 {
		task.Arg2 = &resp.Arg2
	}

	return task, nil
}

// ComputeTask выполняет вычисление для заданной задачи.
func (a *Agent) ComputeTask(task *models.Task) (float64, error) {
	log.Printf("Computing task %d: %f %s %v", task.ID, task.Arg1, task.Operation, task.Arg2)
	if task.Arg2 == nil {
		return task.Arg1, nil
	}
	switch task.Operation {
	case "+":
		return task.Arg1 + *task.Arg2, nil
	case "-":
		return task.Arg1 - *task.Arg2, nil
	case "*":
		return task.Arg1 * *task.Arg2, nil
	case "/":
		if *task.Arg2 == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return task.Arg1 / *task.Arg2, nil
	default:
		return 0, fmt.Errorf("unknown operation")
	}
}

// submitResult отправляет результат выполненной задачи обратно оркестратору через gRPC.
func (a *Agent) submitResult(taskID uint, result float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	req := &pb.TaskResultRequest{
		TaskId: int64(taskID),
		Result: result,
	}

	_, err := a.orchestratorClient.SubmitTaskResult(ctx, req)
	if err != nil {
		return fmt.Errorf("error calling SubmitTaskResult: %w", err)
	}

	log.Printf("Successfully submitted result for task %d: Result=%f", taskID, result)
	return nil
}
