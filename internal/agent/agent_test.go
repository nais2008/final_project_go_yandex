package agent

import (
	"testing"

	"github.com/nais2008/final_project_go_yandex/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestAgent_ComputeTask_Addition(t *testing.T) {
	task := models.Task{
		Arg1:      5,
		Arg2:      ptr(3.0),
		Operation: "+",
	}
	agent := Agent{}
	result := agent.ComputeTask(task)
	assert.Equal(t, 8.0, result)
}

func TestAgent_ComputeTask_Subtraction(t *testing.T) {
	task := models.Task{
		Arg1:      10,
		Arg2:      ptr(4.0),
		Operation: "-",
	}
	agent := Agent{}
	result := agent.ComputeTask(task)
	assert.Equal(t, 6.0, result)
}

func TestAgent_ComputeTask_Multiplication(t *testing.T) {
	task := models.Task{
		Arg1:      7,
		Arg2:      ptr(6.0),
		Operation: "*",
	}
	agent := Agent{}
	result := agent.ComputeTask(task)
	assert.Equal(t, 42.0, result)
}

func TestAgent_ComputeTask_Division(t *testing.T) {
	task := models.Task{
		Arg1:      15,
		Arg2:      ptr(3.0),
		Operation: "/",
	}
	agent := Agent{}
	result := agent.ComputeTask(task)
	assert.Equal(t, 5.0, result)
}

func TestAgent_ComputeTask_DivisionByZero(t *testing.T) {
	task := models.Task{
		Arg1:      10,
		Arg2:      ptr(0.0),
		Operation: "/",
	}
	agent := Agent{}
	result := agent.ComputeTask(task)
	assert.Equal(t, 0.0, result)
}

func TestAgent_ComputeTask_NoArg2(t *testing.T) {
	task := models.Task{
		Arg1:      10,
		Operation: "+",
	}
	agent := Agent{}
	result := agent.ComputeTask(task)
	assert.Equal(t, 10.0, result)
}

func TestAgent_ComputeTask_UnknownOperation(t *testing.T) {
	task := models.Task{
		Arg1:      5,
		Arg2:      ptr(3.0),
		Operation: "%",
	}
	agent := Agent{}
	result := agent.ComputeTask(task)
	assert.Equal(t, 0.0, result)
}

func ptr[T any](v T) *T {
	return &v
}
