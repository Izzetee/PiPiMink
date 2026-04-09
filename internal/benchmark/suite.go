package benchmark

import (
	"context"
	"log"
	"sync"

	"PiPiMink/internal/config"
	"PiPiMink/internal/models"
)

// DB is the minimal database interface required by the benchmark suite.
type DB interface {
	SaveBenchmarkResult(modelName, source, category, taskID string, score float64, latencyMs int64, judgeModel, response string) error
	GetBenchmarkScores(modelName, source string) (map[string]float64, error)
	GetAllBenchmarkScores() (map[string]map[string]float64, error)
}

// Suite orchestrates benchmark runs across all (or a filtered set of) enabled models.
type Suite struct {
	db         DB
	cfg        *config.Config
	scorer     *Scorer
	chatFn     ChatFunc
	tasks      []Task        // if non-nil, overrides AllTasks() for this run
	OnProgress ProgressFunc  // optional callback fired after each task completes
}

// JudgeModel returns the name of the LLM judge model configured for this suite.
func (s *Suite) JudgeModel() string { return s.scorer.judgeModel }

// WithTasks overrides the task list used by this suite. When set, AllTasks() is not called.
// Use this to inject DB-loaded task configs.
func (s *Suite) WithTasks(tasks []Task) *Suite {
	s.tasks = tasks
	return s
}

// NewSuite creates a Suite wired to the given database, config, and chat function.
// The judge provider and model are resolved from BENCHMARK_JUDGE_PROVIDER/MODEL config,
// falling back to MODEL_SELECTION_PROVIDER/MODEL. Per-model overrides are applied.
func NewSuite(db DB, cfg *config.Config, chatFn ChatFunc) *Suite {
	judgeProvider, judgeModel := resolveJudge(cfg)
	return &Suite{
		db:     db,
		cfg:    cfg,
		scorer: NewScorer(judgeProvider, judgeModel),
		chatFn: chatFn,
	}
}

// resolveJudge returns the resolved provider config and model name for the LLM judge.
// It uses BENCHMARK_JUDGE_PROVIDER/MODEL if set, otherwise falls back to the selection provider.
// Per-model overrides (base_url, api_key, type, chat_path) are applied via ForModel().
func resolveJudge(cfg *config.Config) (config.ProviderConfig, string) {
	judgeProviderName := cfg.BenchmarkJudgeProvider
	if judgeProviderName == "" {
		judgeProviderName = cfg.ModelSelectionProvider
	}
	judgeModel := cfg.BenchmarkJudgeModel
	if judgeModel == "" {
		judgeModel = cfg.ModelSelectionModel
	}

	for _, p := range cfg.Providers {
		if p.Name == judgeProviderName {
			return p.ForModel(judgeModel), judgeModel
		}
	}
	return config.ProviderConfig{}, judgeModel
}

// Run executes benchmarks for all provided enabled models.
// Pass categoryFilter = "" to run all categories.
// Pass modelFilter = "" to run all models.
// Models are processed in parallel up to cfg.BenchmarkConcurrency goroutines.
func (s *Suite) Run(ctx context.Context, enabledModels map[string]models.ModelInfo, categoryFilter, modelFilter string) error {
	tasks := s.selectTasks(categoryFilter)
	if len(tasks) == 0 {
		log.Printf("benchmark: no tasks found for category filter %q", categoryFilter)
		return nil
	}

	type workItem struct {
		name string
		info models.ModelInfo
	}

	var work []workItem
	for name, info := range enabledModels {
		if modelFilter != "" && name != modelFilter {
			continue
		}
		work = append(work, workItem{name, info})
	}

	if len(work) == 0 {
		log.Printf("benchmark: no models matched filter %q", modelFilter)
		return nil
	}

	concurrency := s.cfg.BenchmarkConcurrency
	if concurrency <= 0 {
		concurrency = 3
	}

	log.Printf("benchmark: starting run — %d model(s), %d task(s), concurrency=%d", len(work), len(tasks), concurrency)

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, item := range work {
		wg.Add(1)
		sem <- struct{}{}
		go func(name string, info models.ModelInfo) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := s.runForModel(ctx, name, info, tasks); err != nil {
				log.Printf("benchmark: model %s failed: %v", name, err)
			}
		}(item.name, item.info)
	}

	wg.Wait()
	log.Printf("benchmark: run complete")
	return nil
}

// RunForModel runs all tasks (or those in categoryFilter) against a single model.
func (s *Suite) RunForModel(ctx context.Context, modelName string, modelInfo models.ModelInfo, categoryFilter string) error {
	tasks := s.selectTasks(categoryFilter)
	if len(tasks) == 0 {
		return nil
	}
	return s.runForModel(ctx, modelName, modelInfo, tasks)
}

func (s *Suite) runForModel(ctx context.Context, modelName string, modelInfo models.ModelInfo, tasks []Task) error {
	log.Printf("benchmark: running %d task(s) for model %s", len(tasks), modelName)

	results := RunModelTasks(ctx, modelName, modelInfo, tasks, s.scorer, s.chatFn, s.OnProgress)

	for _, r := range results {
		if r.Err != nil {
			continue
		}
		if err := s.db.SaveBenchmarkResult(modelName, modelInfo.Source, string(r.Category), r.TaskID, r.Score, r.LatencyMs, s.JudgeModel(), r.Response); err != nil {
			log.Printf("benchmark: error saving result for model=%s task=%s: %v", modelName, r.TaskID, err)
		}
	}

	// Log per-category averages for this model.
	catScores := make(map[Category][]float64)
	for _, r := range results {
		if r.Err == nil {
			catScores[r.Category] = append(catScores[r.Category], r.Score)
		}
	}
	for cat, scores := range catScores {
		avg := average(scores)
		log.Printf("benchmark: model=%-30s category=%-25s avg=%.2f (%d tasks)", modelName, cat, avg, len(scores))
	}

	return nil
}

// selectTasks returns tasks for this suite run, optionally filtered by category.
// If s.tasks is set it is used as the source; otherwise AllTasks() is called.
func (s *Suite) selectTasks(categoryFilter string) []Task {
	source := s.tasks
	if source == nil {
		source = AllTasks()
	}
	if categoryFilter == "" {
		return source
	}
	var out []Task
	for _, t := range source {
		if string(t.Category) == categoryFilter {
			out = append(out, t)
		}
	}
	return out
}

func average(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
