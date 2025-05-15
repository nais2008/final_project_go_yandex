package parser_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/nais2008/final_project_go_yandex/internal/parser"
)

func TestParseAndCreateTasks_ValidExpression(t *testing.T) {
	tasks, err := parser.ParseAndCreateTasks("3 + 5")
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "+", tasks[0].Operation)
	assert.Equal(t, float64(3), tasks[0].Arg1)
	assert.Equal(t, float64(5), *tasks[0].Arg2)
}

func TestParseAndCreateTasks_InvalidExpression(t *testing.T) {
	_, err := parser.ParseAndCreateTasks("3 + ")
	assert.Error(t, err)
}

func TestSolve_ValidExpression(t *testing.T) {
	result, err := parser.Solve("3 + 5 * 2")
	assert.NoError(t, err)
	assert.Equal(t, float64(13), result)
}

func TestSolve_InvalidExpression(t *testing.T) {
	_, err := parser.Solve("3 + ")
	assert.Error(t, err)
}

func TestIntegration_ParseAndSolve(t *testing.T) {
	tasks, err := parser.ParseAndCreateTasks("10 / 2 + 3 * 4")
	assert.NoError(t, err)
	assert.Len(t, tasks, 3)

	result, err := parser.Solve("10 / 2 + 3 * 4")
	assert.NoError(t, err)
	assert.Equal(t, float64(17), result)
}
