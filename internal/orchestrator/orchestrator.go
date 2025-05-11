package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/models"
	proto "github.com/nais2008/final_project_go_yandex/internal/protos/gen/go/sso"
)

type Orchestrator struct {
	proto.UnimplementedOrchestratorServiceServer
	cfg        config.Config
	db         *gorm.DB
	mutex      sync.Mutex
	authClient proto.AuthServiceClient
}

func NewOrchestrator(cfg config.Config, dbConn *gorm.DB, authConn *grpc.ClientConn) *Orchestrator {
	authClient := proto.NewAuthServiceClient(authConn)
	return &Orchestrator{
		cfg:        cfg,
		db:         dbConn,
		authClient: authClient,
	}
}

func (o *Orchestrator) DB() *gorm.DB {
	return o.db
}

func (o *Orchestrator) authenticate(ctx context.Context) (int64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return 0, status.Error(codes.Unauthenticated, "missing authorization token")
	}

	token := strings.TrimPrefix(values[0], "Bearer ")
	if token == "" {
		return 0, status.Error(codes.Unauthenticated, "invalid token format")
	}

	userID, err := o.getUserIDFromToken(token)
	if err != nil {
		return 0, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	return userID, nil
}

func (o *Orchestrator) getUserIDFromToken(token string) (int64, error) {
	userID, err := strconv.ParseInt(token, 10, 64)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (o *Orchestrator) AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization token"})
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token format"})
			}

			ctx := context.Background()
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
			_, err := o.authClient.Login(ctx, &proto.LoginRequest{})
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}

			return next(c)
		}
	}
}

func (o *Orchestrator) IndexHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "index.html", nil)
}

func (o *Orchestrator) CalculateHandler(c echo.Context) error {
	var req struct {
		Expression string `json:"expression"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "invalid data"})
	}

	ctx := context.Background()
	authHeader := c.Request().Header.Get("Authorization")
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)

	resp, err := o.SubmitExpression(ctx, &proto.ExpressionRequest{Expression: req.Expression})
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"status": resp.Status})
}

func (o *Orchestrator) GetExpressionsHandler(c echo.Context) error {
	var expressions []models.Expression
	if err := o.DB().Find(&expressions).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch expressions"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"expressions": expressions})
}

func (o *Orchestrator) GetExpressionByIDHandler(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid ID"})
	}

	var expr models.Expression
	if err := o.DB().First(&expr, id).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "expression not found"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"expression": expr})
}

func (o *Orchestrator) TaskGetHandler(c echo.Context) error {
	ctx := context.Background()
	authHeader := c.Request().Header.Get("Authorization")
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)

	resp, err := o.GetTask(ctx, &proto.TaskRequest{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if resp.TaskId == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "no tasks available"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"task": map[string]interface{}{
			"id":        resp.TaskId,
			"arg1":      resp.Arg1,
			"arg2":      resp.Arg2,
			"operation": resp.Operation,
		},
	})
}

func (o *Orchestrator) TaskPostHandler(c echo.Context) error {
	var req struct {
		ID     int     `json:"id"`
		Result float64 `json:"result"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "invalid data"})
	}

	ctx := context.Background()
	authHeader := c.Request().Header.Get("Authorization")
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)

	_, err := o.SubmitTaskResult(ctx, &proto.TaskResultRequest{
		TaskId: int64(req.ID),
		Result: req.Result,
	})
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusOK)
}

func (o *Orchestrator) RegisterHandler(c echo.Context) error {
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid data"})
	}

	_, err := o.authClient.Register(context.Background(), &proto.RegisterRequest{
		Email:    req.Login,
		Username: req.Login,
		Password: req.Password,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusOK)
}

func (o *Orchestrator) LoginHandler(c echo.Context) error {
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid data"})
	}

	resp, err := o.authClient.Login(context.Background(), &proto.LoginRequest{
		Login:    req.Login,
		Password: req.Password,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"token": resp.Token})
}

func (o *Orchestrator) SubmitExpression(ctx context.Context, req *proto.ExpressionRequest) (*proto.ExpressionResponse, error) {
	userID, err := o.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	exprStr := strings.ReplaceAll(req.Expression, " ", "")
	if exprStr == "" {
		return nil, status.Error(codes.InvalidArgument, "empty expression")
	}

	tasks, err := o.parseExpression(exprStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid expression: %v", err)
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	var storage models.Storage
	if err := o.db.FirstOrCreate(&storage, models.Storage{NextExprID: 1, NextTaskID: 1}).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to initialize storage: %v", err)
	}

	expr := models.Expression{
		Expr:      req.Expression,
		Status:    "in_progress",
		UserID:    uint(userID),
		Tasks:     tasks,
		StorageID: storage.ID,
	}

	if err := o.db.Create(&expr).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save expression: %v", err)
	}

	storage.NextExprID++
	for range tasks {
		storage.NextTaskID++
	}
	if err := o.db.Save(&storage).Error; err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update storage: %v", err)
	}

	return &proto.ExpressionResponse{
		Status: "in_progress",
	}, nil
}

func (o *Orchestrator) parseExpression(expr string) ([]models.Task, error) {
	numStr := ""
	nums := []float64{}
	ops := []string{}
	tasks := []models.Task{}

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

	if len(nums) < 1 || len(ops) >= len(nums) {
		return nil, errors.New("invalid expression format")
	}

	for i := 0; i < len(ops); {
		if ops[i] == "*" || ops[i] == "/" {
			operationTime := o.getOperationTime(ops[i])
			var storage models.Storage
			if err := o.db.First(&storage).Error; err != nil {
				return nil, err
			}
			task := models.Task{
				ID:            uint(storage.NextTaskID),
				Arg1:          nums[i],
				Arg2:          ptr(nums[i+1]),
				Operation:     ops[i],
				Status:        "pending",
				OperationTime: operationTime,
			}
			tasks = append(tasks, task)
			nums[i] = 0
			nums = append(nums[:i+1], nums[i+2:]...)
			ops = append(ops[:i], ops[i+1:]...)
		} else {
			i++
		}
	}

	for i := 0; i < len(ops); i++ {
		operationTime := o.getOperationTime(ops[i])
		var storage models.Storage
		if err := o.db.First(&storage).Error; err != nil {
			return nil, err
		}
		task := models.Task{
			ID:            uint(storage.NextTaskID),
			Arg1:          nums[i],
			Arg2:          ptr(nums[i+1]),
			Operation:     ops[i],
			Status:        "pending",
			OperationTime: operationTime,
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
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
	}
	return 0
}

func ptr(f float64) *float64 {
	return &f
}

func (o *Orchestrator) GetTask(ctx context.Context, req *proto.TaskRequest) (*proto.TaskResponse, error) {
	userID, err := o.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var expr models.Expression
	if err := o.db.Preload("Tasks").Where("user_id = ? AND status = ?", userID, "in_progress").First(&expr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &proto.TaskResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to query tasks: %v", err)
	}

	for i, task := range expr.Tasks {
		if task.Status == "pending" {
			expr.Tasks[i].Status = "in_progress"
			if err := o.db.Save(&expr).Error; err != nil {
				return nil, status.Errorf(codes.Internal, "failed to update task: %v", err)
			}
			return &proto.TaskResponse{
				TaskId:    int64(task.ID),
				Arg1:      task.Arg1,
				Arg2:      *task.Arg2,
				Operation: task.Operation,
			}, nil
		}
	}

	return &proto.TaskResponse{}, nil
}

func (o *Orchestrator) SubmitTaskResult(ctx context.Context, req *proto.TaskResultRequest) (*proto.TaskResultResponse, error) {
	userID, err := o.authenticate(ctx)
	if err != nil {
		return nil, err
	}

	var expr models.Expression
	if err := o.db.Preload("Tasks").Where("user_id = ?", userID).First(&expr).Error; err != nil {
		return nil, status.Errorf(codes.NotFound, "expression not found: %v", err)
	}

	for i, task := range expr.Tasks {
		if task.ID == uint(req.TaskId) {
			expr.Tasks[i].Result = ptr(req.Result)
			expr.Tasks[i].Status = "completed"
			if err := o.db.Save(&expr).Error; err != nil {
				return nil, status.Errorf(codes.Internal, "failed to save task result: %v", err)
			}
			o.updateExpressionStatus(&expr)
			return &proto.TaskResultResponse{Status: "completed"}, nil
		}
	}

	return nil, status.Error(codes.NotFound, "task not found")
}

func (o *Orchestrator) updateExpressionStatus(expr *models.Expression) {
	allCompleted := true
	for _, task := range expr.Tasks {
		if task.Status != "completed" {
			allCompleted = false
			break
		}
	}
	if allCompleted {
		result := o.computeFinalResult(expr.Tasks)
		expr.Result = ptr(result)
		expr.Status = "completed"
		if err := o.db.Save(expr).Error; err != nil {
			fmt.Printf("failed to update expression status: %v\n", err)
		}
	}
}

func (o *Orchestrator) computeFinalResult(tasks []models.Task) float64 {
	if len(tasks) == 1 && tasks[0].Result != nil {
		return *tasks[0].Result
	}
	result := tasks[0].Arg1
	for i, task := range tasks {
		if task.Result != nil {
			if i > 0 && (task.Operation == "+" || task.Operation == "-") {
				result = *tasks[i-1].Result
			}
			switch task.Operation {
			case "+":
				result += *task.Result
			case "-":
				result -= *task.Result
			case "*":
				result = tasks[0].Arg1 * *task.Result
			case "/":
				result = tasks[0].Arg1 / *task.Result
			}
		}
	}
	return result
}
