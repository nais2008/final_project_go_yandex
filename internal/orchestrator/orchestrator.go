package orchestrator

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/models"
	pb "github.com/nais2008/final_project_go_yandex/internal/protos/gen/go/sso"
	"gorm.io/gorm"
	"google.golang.org/grpc"
)

// Orchestrator - ОСНОВНАЯ СТРУКТУРУ В ДАННОМ ПАКЕТЕ
type Orchestrator struct {
	pb.UnimplementedOrchestratorServiceServer
	cfg                                       config.Config
	gormDB                                    *gorm.DB
	mu                                        sync.Mutex
}

// NewOrchestrator создает основную структуру
func NewOrchestrator(cfg config.Config, gormDB *gorm.DB) *Orchestrator {
	return &Orchestrator{
		cfg:    cfg,
		gormDB: gormDB,
	}
}

// CalculateHandler страница калькулятра )
func (o *Orchestrator) CalculateHandler(c echo.Context) error {
	return o.SubmitExpressionHTTP(c)
}

// SubmitExpressionHTTP ...
func (o *Orchestrator) SubmitExpressionHTTP(ctx echo.Context) error {
	userID := ctx.Get("user_id").(int64)
	var req pb.ExpressionRequest
	if err := ctx.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}
	expression := models.Expression{
		UserID:    uint(userID),
		Expr:      req.Expression,
		Status:    "pending",
		StorageID: 1,
	}
	result := o.gormDB.Create(&expression)
	if result.Error != nil {
		log.Printf("Failed to create expression in DB: %v", result.Error)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to submit expression")
	}
	tasks, err := o.parseExpression(req.Expression, expression.ID)
	if err != nil {
		log.Printf("Failed to parse expression '%s': %v", req.Expression, err)
		expression.Status = "failed"
		o.gormDB.Save(&expression)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid expression")
	}
	for _, task := range tasks {
		res := o.gormDB.Create(&task)
		if res.Error != nil {
			log.Printf("Failed to create task: %v", res.Error)
		}
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{"id": expression.ID, "status": expression.Status})
}

// SubmitExpression gRPC обработчик для получения выражения от других gRPC клиентов (если это необходимо).
func (o *Orchestrator) SubmitExpression(ctx context.Context, req *pb.ExpressionRequest) (*pb.ExpressionResponse, error) {
    log.Printf("Received gRPC expression: '%s'", req.Expression)
    expression := models.Expression{
        Expr:   req.Expression,
        Status: "pending",
        ID:     0, // GORM автоинкремент
    }
    result := o.gormDB.Create(&expression)
    if result.Error != nil {
        return nil, fmt.Errorf("failed to create expression: %w", result.Error)
    }
    return &pb.ExpressionResponse{Status: expression.Status}, nil
}

func (o *Orchestrator) parseExpression(expr string) ([]models.Task, error) {
	expr = strings.ReplaceAll(expr, " ", "")
	if expr == "" {
		return nil, strconv.ErrSyntax
	}

	var tasks []models.Task
	numStr := ""
	nums := []float64{}
	ops := []string{}

	for i := 0; i <= len(expr); i++ {
		if i < len(expr) && (isDigit(expr[i]) || expr[i] == '.') {
			numStr += string(expr[i])
			continue
		}
		if numStr != "" {
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return nil, err
			}
			nums = append(nums, num)
			numStr = ""
		}
		if i < len(expr) && (expr[i] == '+' || expr[i] == '-' || expr[i] == '*' || expr[i] == '/') {
			ops = append(ops, string(expr[i]))
		}
	}

	// Сначала выполняем умножение и деление
	for i := 0; i < len(ops); {
		if ops[i] == "*" || ops[i] == "/" {
			operationTime := o.getOperationTime(ops[i])
			task := models.Task{
				ID:            o.storage.NextTaskID,
				Arg1:          nums[i],
				Arg2:          ptr(nums[i+1]),
				Operation:     ops[i],
				Status:        "pending",
				OperationTime: operationTime,
			}
			o.storage.NextTaskID++
			tasks = append(tasks, task)
			nums[i] = 0 // Пометим как обработанное
			nums = append(nums[:i+1], nums[i+2:]...)
			ops = append(ops[:i], ops[i+1:]...)
		} else {
			i++
		}
	}

	// Затем сложение и вычитание
	for i := 0; i < len(ops); i++ {
		operationTime := o.getOperationTime(ops[i])
		task := models.Task{
			ID:            o.storage.NextTaskID,
			Arg1:          nums[i],
			Arg2:          ptr(nums[i+1]),
			Operation:     ops[i],
			Status:        "pending",
			OperationTime: operationTime,
		}
		o.storage.NextTaskID++
		tasks = append(tasks, task)
	}

	return tasks, nil
}


func (o *Orchestrator) getOperationTime(op string) int {
	switch op {
	case "+":
		return o.cfg.TimeAdditionMS
	case "-":
		return o.cfg.TimeSubtractionMS
	case "*":
		return o.cfg.TimeMultiplicationMS
	case "/":
		return o.cfg.TimeDivisionMS
	default:
		return 1000
	}
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func ptr(f float64) *float64 { return &f }

func (o *Orchestrator) GetTask(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResponse, error) {
	var task models.Task
	result := o.gormDB.Where("status = ?", "pending").First(&task)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Println("GetTask: No pending tasks found.")
			return &pb.TaskResponse{}, nil
		}
		log.Printf("GetTask: Failed to find pending task: %v", result.Error)
		return nil, fmt.Errorf("failed to retrieve task: %w", result.Error)
	}

	log.Printf("GetTask: Found pending task with ID: %d, ExpressionID: %d, Operation: %s", task.ID, task.ExpressionID, task.Operation)

	task.Status = "processing"
	if err := o.gormDB.Save(&task).Error; err != nil {
		log.Printf("GetTask: Failed to update task %d to processing: %v", task.ID, err)
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	resp := &pb.TaskResponse{
		TaskId:        int64(task.ID),
		Arg1:          task.Arg1,
		Operation:     task.Operation,
		OperationTime: int32(task.OperationTime),
	}
	if task.Arg2 != nil {
		resp.Arg2 = *task.Arg2
	}
	return resp, nil
}

func (o *Orchestrator) SubmitTaskResult(ctx context.Context, req *pb.TaskResultRequest) (*pb.TaskResultResponse, error) {
	var task models.Task
	result := o.gormDB.First(&task, req.TaskId)
	if result.Error != nil {
		log.Printf("Failed to find task %d: %v", req.TaskId, result.Error)
		return nil, fmt.Errorf("task not found: %w", result.Error)
	}

	task.Result = &req.Result
	task.Status = "completed"
	o.gormDB.Save(&task)

	// Здесь можно добавить логику проверки завершения всех задач для выражения
	return &pb.TaskResultResponse{Status: "PROCESSED"}, nil
}

func (o *Orchestrator) GetExpressionsHandler(c echo.Context) error {
	userID := c.Get("user_id").(uint)
	var expressions []models.Expression
	result := o.gormDB.Where("user_id = ?", userID).Find(&expressions)
	if result.Error != nil {
		log.Printf("Failed to retrieve expressions for user %d: %v", userID, result.Error)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve expressions")
	}
	return c.JSON(http.StatusOK, expressions)
}

func (o *Orchestrator) GetExpressionByIDHandler(ctx echo.Context) error {
	idStr := ctx.Param("id")
	expressionID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid expression ID")
	}
	var expression models.Expression
	result := o.gormDB.First(&expression, expressionID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Expression not found")
		}
		log.Printf("Failed to find expression: %v", result.Error)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve expression status")
	}
	return ctx.JSON(http.StatusOK, map[string]string{"id": strconv.Itoa(int(expression.ID)), "status": expression.Status, "result": o.formatResult(expression.Result)})
}

func (o *Orchestrator) TaskGetHandler(c echo.Context) error {
	taskResp, err := o.GetTask(c.Request().Context(), &pb.TaskRequest{})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if taskResp == nil || taskResp.TaskId == 0 {
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(http.StatusOK, taskResp)
}

func (o *Orchestrator) TaskPostHandler(c echo.Context) error {
	var req pb.TaskResultRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid task result")
	}
	resp, err := o.SubmitTaskResult(c.Request().Context(), &req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, resp)
}

func (o *Orchestrator) formatResult(result *float64) string {
	if result != nil {
		return strconv.FormatFloat(*result, 'f', 2, 64)
	}
	return ""
}

func (o *Orchestrator) StartGrpcServer() {
	grpcPort := os.Getenv("GRPC_SERVER_ADDRESS")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	listenAddr := fmt.Sprintf(":%s", grpcPort)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOrchestratorServiceServer(grpcServer, o)
	log.Printf("gRPC server listening on %s", listenAddr)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func calculate(a, b float64, op string) float64 {
	switch op {
	case "+":
		return a + b
	case "-":
		return a - b
	case "*":
		return a * b
	case "/":
		if b != 0 {
			return a / b
		}
		return 0
	default:
		return 0
	}
}
