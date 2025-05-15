package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/models"
)

// ParseAndCreateTasks ...
func ParseAndCreateTasks(expr string) ([]models.Task, error) {
	expr = strings.ReplaceAll(expr, " ", "")
	if expr == "" {
		return nil, strconv.ErrSyntax
	}

	tokens, err := tokenize(expr)
	if err != nil {
		return nil, err
	}

	queue, err := infixToPostfix(tokens)
	if err != nil {
		return nil, err
	}

	return createTasksFromPostfix(queue)
}

// Solve ...
func Solve(expr string) (float64, error) {
	expr = strings.ReplaceAll(expr, " ", "")
	if expr == "" {
		return 0, strconv.ErrSyntax
	}

	tokens, err := tokenize(expr)
	if err != nil {
		return 0, err
	}

	queue, err := infixToPostfix(tokens)
	if err != nil {
		return 0, err
	}

	return evaluatePostfix(queue)
}

func tokenize(expression string) ([]string, error) {
	var tokens []string
	var currentToken string
	for _, r := range expression {
		char := string(r)
		if isDigit(r) || char == "." {
			currentToken += char
		} else if isOperator(char) || char == "(" || char == ")" {
			if currentToken != "" {
				tokens = append(tokens, currentToken)
				currentToken = ""
			}
			tokens = append(tokens, char)
		} else if char == " " {
			if currentToken != "" {
				tokens = append(tokens, currentToken)
				currentToken = ""
			}
		} else {
			return nil, fmt.Errorf("недопустимый символ: %s", char)
		}
	}
	if currentToken != "" {
		tokens = append(tokens, currentToken)
	}
	return tokens, nil
}

func infixToPostfix(tokens []string) ([]string, error) {
	var outputQueue []string
	var operatorStack []string
	precedence := map[string]int{
		"+": 1,
		"-": 1,
		"*": 2,
		"/": 2,
	}

	for _, token := range tokens {
		if isNumber(token) {
			outputQueue = append(outputQueue, token)
		} else if isOperator(token) {
			for len(operatorStack) > 0 && isOperator(operatorStack[len(operatorStack)-1]) &&
				precedence[token] <= precedence[operatorStack[len(operatorStack)-1]] {
				outputQueue = append(outputQueue, operatorStack[len(operatorStack)-1])
				operatorStack = operatorStack[:len(operatorStack)-1]
			}

			operatorStack = append(operatorStack, token)
		} else if token == "(" {
			operatorStack = append(operatorStack, token)
		} else if token == ")" {
			for len(operatorStack) > 0 && operatorStack[len(operatorStack)-1] != "(" {
				outputQueue = append(outputQueue, operatorStack[len(operatorStack)-1])
				operatorStack = operatorStack[:len(operatorStack)-1]
			}

			if len(operatorStack) == 0 {
				return nil, fmt.Errorf("несоответствующие скобки")
			}

			operatorStack = operatorStack[:len(operatorStack)-1]
		}
	}

	for len(operatorStack) > 0 {
		if operatorStack[len(operatorStack)-1] == "(" || operatorStack[len(operatorStack)-1] == ")" {
			return nil, fmt.Errorf("несоответствующие скобки")
		}
		outputQueue = append(outputQueue, operatorStack[len(operatorStack)-1])
		operatorStack = operatorStack[:len(operatorStack)-1]
	}

	return outputQueue, nil
}

func evaluatePostfix(tokens []string) (float64, error) {
	var stack []float64

	for _, token := range tokens {
		if isNumber(token) {
			num, err := strconv.ParseFloat(token, 64)
			if err != nil {
				return 0, fmt.Errorf("некорректное число: %s", token)
			}
			stack = append(stack, num)
		} else if isOperator(token) {
			if len(stack) < 2 {
				return 0, fmt.Errorf("недостаточно операндов для оператора: %s", token)
			}

			operand2 := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			operand1 := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			var result float64
			switch token {
			case "+":
				result = operand1 + operand2
			case "-":
				result = operand1 - operand2
			case "*":
				result = operand1 * operand2
			case "/":
				if operand2 == 0 {
					return 0, fmt.Errorf("деление на ноль")
				}
				result = operand1 / operand2
			}
			stack = append(stack, result)
		}
	}

	if len(stack) != 1 {
		return 0, fmt.Errorf("некорректное выражение")
	}
	return stack[0], nil
}

func createTasksFromPostfix(tokens []string) ([]models.Task, error) {
	var stack []float64
	var tasks []models.Task
	var taskCounter int

	for _, token := range tokens {
		if isNumber(token) {
			num, err := strconv.ParseFloat(token, 64)
			if err != nil {
				return nil, fmt.Errorf("некорректное число: %s", token)
			}
			stack = append(stack, num)
		} else if isOperator(token) {
			if len(stack) < 2 {
				return nil, fmt.Errorf("недостаточно операндов для оператора: %s", token)
			}
			operand2 := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			operand1 := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			operationTime := getOperationTime(token)
			task := models.Task{
				Arg1:          operand1,
				Arg2:          ptr(operand2),
				Operation:     token,
				Status:        "pending",
				OperationTime: operationTime,
				Order:         taskCounter,
			}
			tasks = append(tasks, task)
			stack = append(stack, 0)
			taskCounter++
		}
	}

	return tasks, nil
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isOperator(s string) bool {
	return s == "+" || s == "-" || s == "*" || s == "/"
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func getOperationTime(op string) int {
	cfg := config.LoadConfig()

	switch op {
	case "+":
		return cfg.TimeAdditionMS
	case "-":
		return cfg.TimeSubtractionMS
	case "*":
		return cfg.TimeMultiplicationMS
	case "/":
		return cfg.TimeDivisionMS
	default:
		return 0
	}
}

func ptr(f float64) *float64 { return &f }
