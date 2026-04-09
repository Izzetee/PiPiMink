package benchmark

import (
	"context"
	"log"
	"time"

	"PiPiMink/internal/models"
)

// ChatFunc is the function signature for calling a model. It matches llm.Client.ChatWithModel.
type ChatFunc func(modelInfo models.ModelInfo, modelName string, messages []map[string]interface{}) (string, error)

// ProgressFunc is called after each benchmark task completes.
// taskIndex is the 0-based index of the completed task within the current model's run.
type ProgressFunc func(modelName string, taskIndex, totalTasks int, result *TaskResult)

// TaskResult holds the outcome of running a single benchmark task against one model.
type TaskResult struct {
	TaskID    string
	Category  Category
	Score     float64
	LatencyMs int64
	Response  string
	Err       error
}

// RunModelTasks executes the given tasks against a single model and returns one TaskResult per task.
// Tasks are run serially to avoid provider rate-limit issues.
// onProgress is called after each task completes (nil-safe).
func RunModelTasks(
	ctx context.Context,
	modelName string,
	modelInfo models.ModelInfo,
	tasks []Task,
	scorer *Scorer,
	chatFn ChatFunc,
	onProgress ProgressFunc,
) []TaskResult {
	results := make([]TaskResult, 0, len(tasks))

	for i, task := range tasks {
		select {
		case <-ctx.Done():
			log.Printf("benchmark: context cancelled while running tasks for model %s", modelName)
			return results
		default:
		}

		result := runSingleTask(ctx, modelName, modelInfo, task, scorer, chatFn)
		results = append(results, result)

		if onProgress != nil {
			onProgress(modelName, i, len(tasks), &result)
		}
	}

	return results
}

func runSingleTask(
	ctx context.Context,
	modelName string,
	modelInfo models.ModelInfo,
	task Task,
	scorer *Scorer,
	chatFn ChatFunc,
) TaskResult {
	messages := []map[string]interface{}{
		{"role": "user", "content": task.Prompt},
	}

	start := time.Now()
	response, err := chatFn(modelInfo, modelName, messages)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		log.Printf("benchmark: model %s failed task %s: %v", modelName, task.ID, err)
		return TaskResult{
			TaskID:    task.ID,
			Category:  task.Category,
			Score:     0.0,
			LatencyMs: latencyMs,
			Err:       err,
		}
	}

	score := scorer.Score(ctx, task, response)
	log.Printf("benchmark: model=%-30s task=%-40s score=%.2f latency=%dms",
		modelName, task.ID, score, latencyMs)

	return TaskResult{
		TaskID:    task.ID,
		Category:  task.Category,
		Score:     score,
		LatencyMs: latencyMs,
		Response:  response,
	}
}
