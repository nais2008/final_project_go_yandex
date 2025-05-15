package orchestrator

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/db"
	"github.com/nais2008/final_project_go_yandex/internal/models"
	"github.com/nais2008/final_project_go_yandex/internal/parser"
	"gorm.io/gorm"
)

// Orchestrator ...
type Orchestrator struct {
	cfg     config.Config
	storage *db.Storage
}

// NewOrchestrator ...
func NewOrchestrator(cfg config.Config, storage *db.Storage) *Orchestrator {
	return &Orchestrator{cfg: cfg, storage: storage}
}

type calculateRequest struct {
	Expression string `json:"expression"`
}

type calculateResponse struct {
	ID uint `json:"id"`
}

// CalculateHandler ...
func (o *Orchestrator) CalculateHandler(c echo.Context) error {
	userID := c.Get("user_id").(uint)

	var req calculateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "Invalid data"})
	}

	tasks, err := parser.ParseAndCreateTasks(req.Expression)
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": fmt.Sprintf("Invalid expression: %v", err)})
	}

	expr := models.Expression{
		Expr:   req.Expression,
		Status: "in_progress",
		Tasks:  tasks,
		UserID: userID,
	}

	dbResult := o.storage.DB.Create(&expr)
	if dbResult.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save expression with tasks"})
	}

	return c.JSON(http.StatusCreated, calculateResponse{ID: expr.ID})
}

type taskResponse struct {
	Task models.Task `json:"task"`
}

// TaskHandler ...
func (o *Orchestrator) TaskHandler(c echo.Context) error {
	switch c.Request().Method {
	case http.MethodGet:
		var task models.Task
		result := o.storage.DB.Where("status = ?", "pending").First(&task)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "No tasks available"})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch pending task"})
		}
		task.Status = "pending"
		o.storage.DB.Save(&task)
		return c.JSON(http.StatusOK, taskResponse{Task: task})

	case http.MethodPost:
		var req struct {
			ID     uint    `json:"id"`
			Result float64 `json:"result"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "Invalid data"})
		}

		var task models.Task
		if result := o.storage.DB.First(&task, req.ID); result.Error != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Task not found"})
		}

		task.Result = &req.Result
		task.Status = "completed"
		o.storage.DB.Save(&task)

		var expression models.Expression
		o.storage.DB.Preload("Tasks").First(&expression, task.ExpressionID)

		o.updateExpressionStatus(&expression)

		return c.NoContent(http.StatusOK)

	default:
		return c.JSON(http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

func (o *Orchestrator) updateExpressionStatus(expr *models.Expression) {
	if len(expr.Tasks) == 0 {
		o.storage.DB.Model(expr).Updates(map[string]interface{}{"status": "pending", "result": nil})
		return
	}

	completed := true
	for _, task := range expr.Tasks {
		if task.Status != "completed" || task.Result == nil {
			completed = false
			break
		}
	}

	if completed {
		result := o.computeFinalResult(expr.Tasks)
		if result != nil {
			o.storage.DB.Model(expr).Updates(map[string]interface{}{"status": "completed", "result": result})
		} else {
			o.storage.DB.Model(expr).Update("status", "error")
		}
	} else {
		o.storage.DB.Model(expr).Update("status", "in_progress")
	}
}

func (o *Orchestrator) computeFinalResult(tasks []models.Task) *float64 {
	if len(tasks) == 0 {
		return nil
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Order < tasks[j].Order
	})

	var finalResult float64
	firstOperation := true

	for _, task := range tasks {
		if task.Result == nil {
			continue
		}

		if firstOperation {
			finalResult = *task.Result
			firstOperation = false
			continue
		}

		switch task.Operation {
		case "+":
			finalResult += *task.Result
		case "-":
			finalResult -= *task.Result
		case "*":
			finalResult *= *task.Result
		case "/":
			if *task.Result == 0 {
				return nil
			}
			finalResult /= *task.Result
		default:
			return nil
		}
	}

	if firstOperation {
		return nil
	}

	return &finalResult
}

type expressionsResponse struct {
	Expressions []models.Expression `json:"expressions"`
}

// GetExpressionsHandler ...
func (o *Orchestrator) GetExpressionsHandler(c echo.Context) error {
	userID := c.Get("user_id").(uint)
	
	var expressions []models.Expression
	result := o.storage.DB.Where("user_id = ?", userID).Preload("Tasks").Find(&expressions)

	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch expressions"})
	}

	return c.JSON(http.StatusOK, expressionsResponse{Expressions: expressions})
}

type expressionResponse struct {
	Expression models.Expression `json:"expression"`
}

// GetExpressionByIDHandler ...
func (o *Orchestrator) GetExpressionByIDHandler(c echo.Context) error {
	userID := c.Get("user_id").(uint)
	idStr := c.Param("id")

	if idStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ID is required"})
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
	}

	var expression models.Expression
	result := o.storage.DB.Where("id = ? AND user_id = ?", id, userID).Preload("Tasks").First(&expression)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Expression not found"})
		}

		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch expression"})
	}

	return c.JSON(http.StatusOK, expressionResponse{Expression: expression})
}
