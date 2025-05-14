package agent

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/nais2008/final_project_go_yandex/internal/config"
	"github.com/nais2008/final_project_go_yandex/internal/db"
	"github.com/nais2008/final_project_go_yandex/internal/models"
)

type Agent struct {
	cfg              config.Config
	s                *db.Storage
	orchestratorAddr string
}

func NewAgent(cfg config.Config) *Agent {
	st, err := db.ConnectDB()
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	return &Agent{cfg: cfg, s: st, orchestratorAddr: cfg.OrchestratorAddr}
}

func (a *Agent) Run() {
	for {
		task, err := a.getTask()
		if err != nil {
			log.Printf("Error getting task: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if task.ID == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		// Проверяем наличие Expression в базе данных
		var expr models.Expression
		if err := a.s.DB.First(&expr, task.ExpressionID).Error; err != nil {
			log.Printf("Expression not found for task %d: %v", task.ID, err)
			continue
		}

		// Проверяем статус задачи, если не "pending", то пропускаем её
		if task.Status != "pending" {
			log.Printf("Skipping task %d as it is not in 'pending' status", task.ID)
			continue
		}

		// Вычисляем результат
		result := a.ComputeTask(task)
		task.Result = &result
		task.Status = "completed"

		// Обновляем задачу в базе данных с результатом
		if err := a.s.DB.Save(&task).Error; err != nil {
			log.Printf("Error saving task %d with result: %v", task.ID, err)
			continue
		}

		// Ждём указанное время
		time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)

		a.submitResult(task.ID, result)
	}
}

func (a *Agent) getTask() (models.Task, error) {
    resp, err := http.Get("http://" + a.orchestratorAddr + "/internal/tasks")  // изменено на /internal/tasks
    if err != nil || resp.StatusCode != http.StatusOK {
        return models.Task{}, err
    }
    defer resp.Body.Close()

    var data struct {
        Task models.Task `json:"task"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return models.Task{}, err
    }

    return data.Task, nil
}

func (a *Agent) ComputeTask(task models.Task) float64 {
	if task.Arg2 == nil {
		return task.Arg1
	}
	switch task.Operation {
	case "+":
		return task.Arg1 + *task.Arg2
	case "-":
		return task.Arg1 - *task.Arg2
	case "*":
		return task.Arg1 * *task.Arg2
	case "/":
		if *task.Arg2 == 0 {
			return 0
		}
		return task.Arg1 / *task.Arg2
	default:
		return 0
	}
}

func (a *Agent) submitResult(taskID uint, result float64) {
	payload := map[string]interface{}{
		"id":     taskID,
		"result": result,
	}
	body, _ := json.Marshal(payload)

	_, err := http.Post("http://" + a.orchestratorAddr + "/internal/task", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error submitting result for task %d: %v", taskID, err)
	}
}
