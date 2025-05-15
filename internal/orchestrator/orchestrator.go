package orchestrator

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/db"
	"github.com/nais2008/final_project_go_yandex/internal/models"
	"gorm.io/gorm"
)

type Orchestrator struct {
	cfg     config.Config
	storage *db.Storage
}

func NewOrchestrator(cfg config.Config, storage *db.Storage) *Orchestrator {
	return &Orchestrator{cfg: cfg, storage: storage}
}

type calculateRequest struct {
	Expression string `json:"expression"`
}

type calculateResponse struct {
	ID uint `json:"id"`
}

func (o *Orchestrator) CalculateHandler(c echo.Context) error {
	userID := c.Get("user_id").(uint)

	var req calculateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "Invalid data"})
	}

	tasks, err := o.parseExpression(req.Expression)
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "Invalid expression"})
	}

	expr := models.Expression{
		Expr:   req.Expression,
		Status: "in_progress",
		Tasks:  tasks,
		UserID: userID,
	}

	result := o.storage.DB.Create(&expr)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save expression"})
	}

	return c.JSON(http.StatusCreated, calculateResponse{ID: expr.ID})
}

func (o *Orchestrator) parseExpression(expr string) ([]models.Task, error) {
	expr = strings.ReplaceAll(expr, " ", "")

	if expr == "" {
		return nil, strconv.ErrSyntax
	}

	isDigit := func(ch byte) bool {
		return (ch >= '0' && ch <= '9')
	}

	numStr := ""
	nums := []float64{}
	ops := []string{}

	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if isDigit(ch) || ch == '.' {
			numStr += string(ch)
		} else if ch == '+' || ch == '-' || ch == '*' || ch == '/' {
			if numStr == "" {
				return nil, fmt.Errorf("ошибка парсинга: ожидается число перед оператором в позиции %d", i)
			}
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return nil, err
			}
			nums = append(nums, num)
			numStr = ""
			ops = append(ops, string(ch))
		} else {
			return nil, fmt.Errorf("неизвестный символ: %c", ch)
		}
	}
	if numStr == "" {
		return nil, fmt.Errorf("ошибка парсинга: выражение заканчивается на оператор")
	}
	lastNum, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil, err
	}
	nums = append(nums, lastNum)

	tasks := make([]models.Task, 0)

	// Первый проход: умножение и деление
	tempNums := make([]float64, 0)
	tempNums = append(tempNums, nums[0])
	for i, op := range ops {
		if op == "*" || op == "/" {
			operationTime := o.getOperationTime(op)
			arg1 := tempNums[len(tempNums)-1]
			arg2 := nums[i+1]
			tasks = append(tasks, models.Task{
				Arg1:          arg1,
				Arg2:          ptr(arg2),
				Operation:     op,
				Status:        "pending",
				OperationTime: operationTime,
			})
			// Здесь мы *не* выполняем операцию, а просто создаем задачу.
			// Результат будет обработан позже оркестратором.
			tempNums = append(tempNums[:len(tempNums)-1], 0) // Заглушка, фактическое значение не важно
			tempNums = append(tempNums, 0)                 // Заглушка
		} else {
			tempNums = append(tempNums, nums[i+1])
		}
	}

	// Второй проход: сложение и вычитание
	newNums := make([]float64, 0)
	newNums = append(newNums, nums[0])
	newOps := make([]string, 0)
	for i, op := range ops {
		if op == "+" || op == "-" {
			newNums = append(newNums, nums[i+1])
			newOps = append(newOps, op)
		} else {
			// Пропускаем * и /, они уже обработаны
		}
	}

	// Создаем задачи для сложения и вычитания
	for i, op := range newOps {
		operationTime := o.getOperationTime(op)
		arg1 := newNums[i] // Обратите внимание на индексы
		arg2 := newNums[i+1]
		tasks = append(tasks, models.Task{
			Arg1:          arg1,
			Arg2:          ptr(arg2),
			Operation:     op,
			Status:        "pending",
			OperationTime: operationTime,
		})
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
	}
	return 0
}

func ptr(f float64) *float64 { return &f }

type taskResponse struct {
	Task models.Task `json:"task"`
}

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
		o.storage.DB.Model(expr).Update("status", "pending").Update("result", nil)
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
		o.storage.DB.Model(expr).Updates(map[string]interface{}{
			"status": "completed",
			"result": result,
		})
	} else {
		o.storage.DB.Model(expr).Update("status", "in_progress")
	}
}


func (o *Orchestrator) computeFinalResult(tasks []models.Task) float64 {
	result := tasks[0].Arg1

	for _, task := range tasks {
		if task.Result != nil {
			switch task.Operation {
			case "+":
				result += *task.Result
			case "-":
				result -= *task.Result
			case "*":
				result *= *task.Result
			case "/":
				if *task.Result != 0 {
					result /= *task.Result
				}
			}
		}
	}

	return result
}

type expressionsResponse struct {
	Expressions []models.Expression `json:"expressions"`
}

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
