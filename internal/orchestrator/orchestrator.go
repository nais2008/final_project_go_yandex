package orchestrator

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/models"
	"github.com/nais2008/final_project_go_yandex/internal/protos/gen/go/sso"
	"gorm.io/gorm"
)

type Orchestrator struct {
	cfg    config.Config
	gormDB *gorm.DB
	mu     sync.Mutex
}

func NewOrchestrator(cfg config.Config, gormDB *gorm.DB) *Orchestrator {
	return &Orchestrator{
		cfg:    cfg,
		gormDB: gormDB,
	}
}

func (o *Orchestrator) CalculateHandler(c echo.Context) error {
	return o.SubmitExpression(c)
}

func (o *Orchestrator) SubmitExpression(ctx echo.Context) error {
	userID := ctx.Get("user_id").(int64)
	var req proto.ExpressionRequest
	if err := ctx.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}

	log.Printf("Received expression from request: '%s'", req.Expression) // Логируем полученное выражение

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

	log.Printf("Expression created in DB with ID: %d, Expression: '%s'", expression.ID, expression.Expr) // Логируем созданное выражение в БД

	tasks, err := o.parseExpression(req.Expression, expression.ID)
	if err != nil {
		log.Printf("Failed to parse expression '%s': %v", req.Expression, err)
		expression.Status = "failed"
		o.gormDB.Save(&expression)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid expression")
	}

	log.Printf("Parsed tasks for expression %d: %v", expression.ID, tasks) // Логируем спарсенные задачи

	for _, task := range tasks {
		res := o.gormDB.Create(&task)
		if res.Error != nil {
			log.Printf("Failed to create task: %v", res.Error)
		} else {
			log.Printf("Created task for expression %d: %f %s %f (ID: %d)", expression.ID, task.Arg1, task.Operation, *task.Arg2, task.ID)
		}
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"id": expression.ID, "status": expression.Status})
}

func (o *Orchestrator) parseExpression(expr string, expressionID uint) ([]models.Task, error) {
	expr = strings.ReplaceAll(expr, " ", "")
	log.Printf("Parsing expression: '%s'", expr) // Логируем входное выражение
	if expr == "" {
		return nil, strconv.ErrSyntax
	}

	var tasks []models.Task
	var numbers []float64
	var operators []string
	currentNumber := ""

	for i, char := range expr {
		log.Printf("Processing char at index %d: '%s'", i, string(char)) // Логируем каждый символ
		if isDigit(byte(char)) || char == '.' {
			currentNumber += string(char)
		} else if char == '+' || char == '-' || char == '*' || char == '/' {
			if currentNumber != "" {
				num, err := strconv.ParseFloat(currentNumber, 64)
				if err != nil {
					return nil, err
				}
				numbers = append(numbers, num)
				log.Printf("Parsed number: %f, numbers: %v", num, numbers) // Логируем спарсенное число
				currentNumber = ""
			}
			operators = append(operators, string(char))
			log.Printf("Found operator: '%s', operators: %v", string(char), operators) // Логируем оператор
		} else {
			log.Printf("Invalid character: '%s'", string(char)) // Логируем невалидный символ
			return nil, strconv.ErrSyntax // Invalid character in expression
		}
	}
	if currentNumber != "" {
		num, err := strconv.ParseFloat(currentNumber, 64)
		if err != nil {
			return nil, err
		}
		numbers = append(numbers, num)
		log.Printf("Parsed final number: %f, numbers: %v", num, numbers) // Логируем последнее число
	}

	log.Printf("Final numbers: %v, Final operators: %v", numbers, operators) // Логируем итоговые слайсы

	if len(numbers) != len(operators)+1 {
		log.Printf("Number of numbers (%d) does not equal number of operators + 1 (%d)", len(numbers), len(operators)+1)
		return nil, strconv.ErrSyntax // Invalid number of operands/operators
	}

	// Helper function to apply an operation and create a task
	applyOpAndCreateTask := func(op string, arg1, arg2 float64) {
		operationTime := o.getOperationTime(op)
		task := models.Task{
			ExpressionID:  expressionID,
			Arg1:          arg1,
			Arg2:          ptr(arg2),
			Operation:     op,
			Status:        "pending",
			OperationTime: operationTime,
		}
		tasks = append(tasks, task)
		log.Printf("Created task: %v", task) // Логируем созданную задачу
	}

	// First pass: Multiplication and Division
	newNumbers := make([]float64, 0)
	newNumbers = append(newNumbers, numbers[0])
	newOperators := make([]string, 0)

	for i := 0; i < len(operators); i++ {
		op := operators[i]
		if op == "*" || op == "/" {
			arg1 := newNumbers[len(newNumbers)-1]
			arg2 := numbers[i+1]
			applyOpAndCreateTask(op, arg1, arg2)
			result := calculate(arg1, arg2, op)
			newNumbers[len(newNumbers)-1] = result
		} else {
			newNumbers = append(newNumbers, numbers[i+1])
			newOperators = append(newOperators, op)
		}
	}

	numbers = newNumbers
	operators = newOperators

	// Second pass: Addition and Subtraction
	for i := 0; i < len(operators); i++ {
		op := operators[i]
		arg1 := numbers[i]
		arg2 := numbers[i+1]
		applyOpAndCreateTask(op, arg1, arg2)
		numbers[i] = calculate(arg1, arg2, op)
	}

	log.Printf("Parsed tasks: %v", tasks) // Логируем итоговый слайс задач
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

func (o *Orchestrator) GetTaskForAgent(ctx echo.Context) error {
	var task models.Task
	result := o.gormDB.Where("status = ?", "pending").First(&task)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return ctx.NoContent(http.StatusNoContent)
		}
		log.Printf("Failed to find pending task: %v", result.Error)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve task")
	}
	task.Status = "processing"
	o.gormDB.Save(&task)
	return ctx.JSON(http.StatusOK, &proto.TaskResponse{
		TaskId:    int64(task.ID),
		Arg1:      task.Arg1,
		Arg2:      *task.Arg2,
		Operation: task.Operation,
		OperationTime: int32(task.OperationTime),
	})
}

func (o *Orchestrator) SubmitTaskResultFromAgent(ctx echo.Context) error {
	var req proto.TaskResultRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid task result")
	}

	var task models.Task
	result := o.gormDB.First(&task, req.TaskId)
	if result.Error != nil {
		log.Printf("Failed to find task %d: %v", req.TaskId, result.Error)
		return echo.NewHTTPError(http.StatusNotFound, "Task not found")
	}

	task.Result = &req.Result
	task.Status = "completed"
	o.gormDB.Save(&task)

	var expression models.Expression
	o.gormDB.Preload("Tasks").First(&expression, task.ExpressionID)
	allCompleted := true
	finalResult := 0.0
	for _, t := range expression.Tasks {
		if t.Status != "completed" {
			allCompleted = false
			break
		}
		if t.Result != nil {
			finalResult = *t.Result
		}
	}

	if allCompleted {
		expression.Status = "completed"
		expression.Result = &finalResult
		o.gormDB.Save(&expression)
		log.Printf("Expression %d completed with result: %f", expression.ID, finalResult)
	}

	return ctx.JSON(http.StatusOK, proto.TaskResultResponse{Status: "PROCESSED"})
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
	return o.GetTaskForAgent(c)
}

func (o *Orchestrator) TaskPostHandler(c echo.Context) error {
	return o.SubmitTaskResultFromAgent(c)
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

func (o *Orchestrator) formatResult(result *float64) string {
	if result != nil {
		return strconv.FormatFloat(*result, 'f', 2, 64)
	}
	return ""
}
