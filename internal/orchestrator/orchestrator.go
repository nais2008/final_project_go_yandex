package orchestrator

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/models"
	"github.com/nais2008/final_project_go_yandex/proto"
	"gorm.io/gorm"
)

type Orchestrator struct {
	db  *gorm.DB
	cfg config.Config
}

func NewOrchestrator(db *gorm.DB, cfg config.Config) *Orchestrator {
	return &Orchestrator{db: db, cfg: cfg}
}

func (o *Orchestrator) SubmitExpression(ctx context.Context, req *proto.ExpressionRequest) (*proto.ExpressionResponse, error) {
	expr := models.Expression{Expr: req.GetExpression(), Status: "pending"}
	if err := o.db.Create(&expr).Error; err != nil {
		return nil, fmt.Errorf("create expression: %w", err)
	}
	tasks := o.createTasks(&expr)
	expr.Tasks = tasks
	o.db.Save(&expr)
	return &proto.ExpressionResponse{Status: "queued", Result: strconv.FormatUint(uint64(expr.ID), 10)}, nil
}

func (o *Orchestrator) GetTask(ctx context.Context, req *proto.TaskRequest) (*proto.TaskResponse, error) {
	var task models.Task
	if err := o.db.Where("status = ?", "pending").Order("id").First(&task).Error; err != nil {
		return nil, fmt.Errorf("no pending task: %w", err)
	}
	task.Status = "in_progress"
	o.db.Save(&task)
	return &proto.TaskResponse{TaskId: int64(task.ID), Arg1: task.Arg1, Arg2: *task.Arg2, Operation: task.Operation}, nil
}

func (o *Orchestrator) SubmitTaskResult(ctx context.Context, req *proto.TaskResultRequest) (*proto.TaskResultResponse, error) {
	var task models.Task
	if err := o.db.First(&task, req.GetTaskId()).Error; err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}
	task.Result = &req.Result
	task.Status = "completed"
	o.db.Save(&task)
	o.updateExpressionStatus(task.ExpressionID)
	return &proto.TaskResultResponse{Status: "ok"}, nil
}

func (o *Orchestrator) RegisterRoutes(e *echo.Echo) {
	e.POST("/api/v1/expressions", o.httpSubmitExpression)
	e.GET("/api/v1/expressions", o.httpListExpressions)
	e.GET("/api/v1/expressions/:id", o.httpGetExpression)
}

func (o *Orchestrator) httpSubmitExpression(c echo.Context) error {
	var body struct{ Expression string }
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	rpcResp, err := o.SubmitExpression(c.Request().Context(), &proto.ExpressionRequest{Expression: body.Expression})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, rpcResp)
}

func (o *Orchestrator) httpListExpressions(c echo.Context) error {
	var exprs []models.Expression
	o.db.Preload("Tasks").Find(&exprs)
	return c.JSON(http.StatusOK, exprs)
}

func (o *Orchestrator) httpGetExpression(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var expr models.Expression
	if err := o.db.Preload("Tasks").First(&expr, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "expression not found")
	}
	return c.JSON(http.StatusOK, expr)
}

func (o *Orchestrator) createTasks(expr *models.Expression) []models.Task {
	return []models.Task{}
}

func (o *Orchestrator) updateExpressionStatus(exprID uint) {
	var tasks []models.Task
	o.db.Where("expression_id = ?", exprID).Find(&tasks)
	allDone := true
	for _, t := range tasks {
		if t.Status != "completed" {
			allDone = false
			break
		}
	}
	if allDone {
		o.db.Model(&models.Expression{}).
			Where("id = ?", exprID).
			Updates(map[string]interface{}{"status": "completed"})
	}
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
