// Package server provides model management utilities for the server
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"PiPiMink/internal/benchmark"
	"PiPiMink/internal/config"
	"PiPiMink/internal/models"
)

// ModelTarget identifies a single model by name and provider.
type ModelTarget struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

// hasUsefulTags returns true if the tags JSON contains at least one strength.
// Models that responded with {} or an empty strengths array are not usable for routing.
func hasUsefulTags(tags string) bool {
	if tags == "" || tags == "{}" {
		return false
	}
	var t map[string]interface{}
	if err := json.Unmarshal([]byte(tags), &t); err != nil {
		return false
	}
	strengths, ok := t["strengths"].([]interface{})
	return ok && len(strengths) > 0
}

// parseTags parses the raw tags JSON string into a structured map with
// "strengths" and "weaknesses" string arrays. Returns a safe default if parsing fails.
func parseTags(tags string) map[string][]string {
	result := map[string][]string{"strengths": {}, "weaknesses": {}}
	if tags == "" || tags == "{}" {
		return result
	}
	var raw struct {
		Strengths  []string `json:"strengths"`
		Weaknesses []string `json:"weaknesses"`
	}
	if err := json.Unmarshal([]byte(tags), &raw); err != nil {
		return result
	}
	if raw.Strengths != nil {
		result["strengths"] = raw.Strengths
	}
	if raw.Weaknesses != nil {
		result["weaknesses"] = raw.Weaknesses
	}
	return result
}

// loadModelsFromDatabase loads models from the database into the model collection
// and enriches each ModelInfo with its benchmark scores.
func (s *Server) loadModelsFromDatabase() error {
	log.Println("Loading models from database")

	dbModels, err := s.db.GetAllModels()
	if err != nil {
		return fmt.Errorf("error getting models from database: %w", err)
	}

	// Load benchmark scores and latencies to enrich ModelInfo entries.
	allScores, scoresErr := s.db.GetAllBenchmarkScores()
	if scoresErr != nil {
		log.Printf("Warning: could not load benchmark scores: %v", scoresErr)
		allScores = nil
	}

	allLatencies, latErr := s.db.GetAllModelLatencies()
	if latErr != nil {
		log.Printf("Warning: could not load model latencies: %v", latErr)
		allLatencies = nil
	}

	s.modelMutex.Lock()
	defer s.modelMutex.Unlock()

	newCollection := models.NewModelCollection()
	newCollection.FromDatabaseMap(dbModels)

	// Attach benchmark scores and latencies to each model.
	for name, info := range newCollection.Models {
		key := name + "|" + info.Source
		changed := false
		if allScores != nil {
			if scores, ok := allScores[key]; ok && len(scores) > 0 {
				info.BenchmarkScores = scores
				changed = true
			}
		}
		if allLatencies != nil {
			if ms, ok := allLatencies[key]; ok {
				info.AvgLatencyMs = &ms
				changed = true
			}
		}
		if changed {
			newCollection.UpdateModel(name, info)
		}
	}

	s.modelCollection = newCollection
	log.Printf("Loaded %d models from database", len(s.modelCollection.Models))
	return nil
}

// discoverModels queries every configured provider for its model list and registers each model
// in the database (no-op if already present). Returns the total number of newly registered models.
func (s *Server) discoverModels() (providers int, discovered int, err error) {
	// Copy providers under lock so concurrent CRUD doesn't race.
	s.providerMutex.RLock()
	providersCopy := make([]config.ProviderConfig, len(s.config.Providers))
	copy(providersCopy, s.config.Providers)
	s.providerMutex.RUnlock()

	log.Printf("Discovering models from %d configured provider(s)", len(providersCopy))

	for _, provider := range providersCopy {
		if !provider.Enabled {
			continue
		}
		names, listErr := s.llmClient.GetModelsByProvider(provider)
		if listErr != nil {
			log.Printf("discover: error listing models from provider %s: %v", provider.Name, listErr)
			continue
		}
		providers++
		for _, name := range names {
			if regErr := s.db.RegisterDiscoveredModel(name, provider.Name); regErr != nil {
				log.Printf("discover: error registering model %s (%s): %v", name, provider.Name, regErr)
				continue
			}
			discovered++
		}
		log.Printf("discover: registered %d model(s) from provider %s", len(names), provider.Name)
	}

	// Reload in-memory collection so the admin UI can see newly discovered models.
	if reloadErr := s.loadModelsFromDatabase(); reloadErr != nil {
		log.Printf("discover: error reloading models: %v", reloadErr)
	}

	return providers, discovered, nil
}

// loadTaggingPromptsFromDB reads the three tagging prompt overrides from the database
// and pushes them to the LLM client. Called before every tagging run so prompt edits
// in the admin UI take effect immediately without a restart.
func (s *Server) loadTaggingPromptsFromDB() {
	keys := []string{"tagging_system", "tagging_user", "tagging_user_nosys"}
	values := make([]string, 3)
	for i, key := range keys {
		if val, found, err := s.db.GetSystemPrompt(key); err != nil {
			log.Printf("tag: could not load prompt %q from DB: %v", key, err)
		} else if found {
			values[i] = val
		}
	}
	s.llmClient.UpdateTaggingPrompts(values[0], values[1], values[2])
}

// tagModels runs GetModelTags for each supplied (name, source) pair and persists the result.
// Unknown provider names are skipped with a log message.
func (s *Server) tagModels(targets []ModelTarget) {
	// Initialise operation tracker.
	s.tagOpMutex.Lock()
	s.tagOperation = &operationState{
		Status:    "running",
		Total:     len(targets),
		Completed: 0,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.tagOpMutex.Unlock()

	// Apply any prompt overrides stored in the DB before tagging.
	s.loadTaggingPromptsFromDB()

	// Build provider lookup under lock.
	s.providerMutex.RLock()
	providerByName := make(map[string]config.ProviderConfig, len(s.config.Providers))
	for _, p := range s.config.Providers {
		providerByName[p.Name] = p
	}
	s.providerMutex.RUnlock()

	for i, t := range targets {
		// Update current model before starting.
		s.tagOpMutex.Lock()
		s.tagOperation.CurrentModel = t.Name
		s.tagOpMutex.Unlock()

		p, ok := providerByName[t.Source]
		if !ok {
			log.Printf("tag: unknown provider %q for model %s — skipping", t.Source, t.Name)
			s.tagOpMutex.Lock()
			s.tagOperation.Completed = i + 1
			s.tagOperation.FailedModels = append(s.tagOperation.FailedModels, t.Name)
			s.tagOpMutex.Unlock()
			continue
		}

		tags, shouldDisable, shouldDelete, err := s.llmClient.GetModelTags(t.Name, p)
		if err != nil {
			log.Printf("tag: error tagging model %s (%s): %v", t.Name, t.Source, err)
			tags = "{}"
			s.tagOpMutex.Lock()
			s.tagOperation.FailedModels = append(s.tagOperation.FailedModels, t.Name)
			s.tagOpMutex.Unlock()
		}
		if shouldDelete {
			log.Printf("tag: deleting non-chat model %s (%s)", t.Name, t.Source)
			if delErr := s.db.DeleteModel(t.Name, t.Source); delErr != nil {
				log.Printf("tag: delete error: %v", delErr)
			}
			s.tagOpMutex.Lock()
			s.tagOperation.Completed = i + 1
			s.tagOpMutex.Unlock()
			continue
		}
		enabled := !shouldDisable && hasUsefulTags(tags)
		hasReasoning := models.IsReasoningModel(t.Name)
		if saveErr := s.db.SaveModel(t.Name, t.Source, tags, enabled, hasReasoning); saveErr != nil {
			log.Printf("tag: error saving model %s: %v", t.Name, saveErr)
		}
		log.Printf("tag: tagged %s (%s) enabled=%v", t.Name, t.Source, enabled)

		s.tagOpMutex.Lock()
		s.tagOperation.Completed = i + 1
		s.tagOpMutex.Unlock()
	}

	if reloadErr := s.loadModelsFromDatabase(); reloadErr != nil {
		log.Printf("tag: error reloading models: %v", reloadErr)
	}

	// Mark operation as completed.
	s.tagOpMutex.Lock()
	s.tagOperation.Status = "completed"
	s.tagOperation.CurrentModel = ""
	s.tagOpMutex.Unlock()
}

// fetchAndTagModels fetches models from all configured providers and tags them with capabilities.
func (s *Server) fetchAndTagModels() error {
	// Copy providers under lock so concurrent CRUD doesn't race.
	s.providerMutex.RLock()
	providersCopy := make([]config.ProviderConfig, len(s.config.Providers))
	copy(providersCopy, s.config.Providers)
	s.providerMutex.RUnlock()

	log.Printf("Fetching and tagging models from %d configured provider(s)", len(providersCopy))

	var wg sync.WaitGroup
	var mu sync.Mutex
	allModels := make(map[string]models.ModelInfo)

	for _, provider := range providersCopy {
		if !provider.Enabled {
			continue
		}
		wg.Add(1)
		go func(p config.ProviderConfig) {
			defer wg.Done()

			modelNames, err := s.llmClient.GetModelsByProvider(p)
			if err != nil {
				log.Printf("Error getting models from provider %s: %v", p.Name, err)
				return
			}

			log.Printf("Found %d model(s) from provider %s", len(modelNames), p.Name)

			for _, name := range modelNames {
				tags, shouldDisable, shouldDelete, err := s.llmClient.GetModelTags(name, p)
				if err != nil {
					log.Printf("Error getting tags for model %s (provider %s): %v", name, p.Name, err)
					tags = "{}"
				}

				if shouldDelete {
					log.Printf("Model %s (provider %s) is not a chat model and will be deleted", name, p.Name)
					if err := s.db.DeleteModel(name, p.Name); err != nil {
						log.Printf("Error deleting model %s: %v", name, err)
					}
					continue
				}

				// Disable models that are flagged incompatible OR returned no useful capability tags.
				// Empty tags ({}) mean the model did not self-report strengths — it cannot be routed.
				enabled := !shouldDisable && hasUsefulTags(tags)
				if !enabled && !shouldDisable {
					log.Printf("Model %s (provider %s) disabled: no capability tags returned", name, p.Name)
				}
				hasReasoning := models.IsReasoningModel(name)

				if err := s.db.SaveModel(name, p.Name, tags, enabled, hasReasoning); err != nil {
					log.Printf("Error saving model %s to database: %v", name, err)
					continue
				}

				mu.Lock()
				allModels[name] = models.ModelInfo{
					Source:       p.Name,
					Tags:         tags,
					Enabled:      enabled,
					HasReasoning: hasReasoning,
					UpdatedAt:    time.Now().Format(time.RFC3339),
				}
				mu.Unlock()

				log.Printf("Tagged and saved model %s (provider: %s, enabled: %v)", name, p.Name, enabled)
			}
		}(provider)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Update the model collection with the new models
	s.modelMutex.Lock()
	defer s.modelMutex.Unlock()

	for name, info := range allModels {
		s.modelCollection.AddModel(name, info)
	}

	return nil
}

// logModels logs the currently loaded models for debugging
func (s *Server) logModels() {
	s.modelMutex.RLock()
	defer s.modelMutex.RUnlock()

	log.Printf("=== Currently loaded models (%d) ===", len(s.modelCollection.Models))
	for name, info := range s.modelCollection.Models {
		log.Printf("- %s (source: %s, enabled: %v, updated: %s)",
			name, info.Source, info.Enabled, info.UpdatedAt)
	}
	log.Println("=== End of model list ===")
}

// loadBenchmarkTasksFromDB loads enabled task configs from the DB and converts them to []Task.
// resolveBenchmarkJudgeName returns the configured benchmark judge model name.
// Falls back to the model selection model if no dedicated judge is configured.
func (s *Server) resolveBenchmarkJudgeName() string {
	if s.config.BenchmarkJudgeModel != "" {
		return s.config.BenchmarkJudgeModel
	}
	return s.config.ModelSelectionModel
}

// Falls back to the compiled-in defaults when the DB returns no rows.
func (s *Server) loadBenchmarkTasksFromDB() []benchmark.Task {
	cfgs, err := s.db.GetBenchmarkTaskConfigs()
	if err != nil {
		log.Printf("benchmark: could not load task configs from DB, using defaults: %v", err)
		return benchmark.AllTasks()
	}
	if len(cfgs) == 0 {
		log.Println("benchmark: no task configs in DB, using defaults")
		return benchmark.AllTasks()
	}
	tasks := benchmark.TasksFromConfigs(cfgs)
	log.Printf("benchmark: loaded %d task(s) from DB", len(tasks))
	return tasks
}

// runBenchmarks executes the benchmark suite against enabled models and reloads scores.
// modelFilter and categoryFilter are optional — pass "" to run all.
func (s *Server) runBenchmarks(ctx context.Context, modelFilter, categoryFilter string) {
	if !s.config.BenchmarkEnabled {
		log.Println("benchmark: disabled via config — skipping")
		return
	}

	s.modelMutex.RLock()
	enabledModels := s.modelCollection.GetEnabledModels()
	s.modelMutex.RUnlock()

	if len(enabledModels) == 0 {
		log.Println("benchmark: no enabled models — skipping")
		return
	}

	chatFn := benchmark.ChatFunc(func(modelInfo models.ModelInfo, modelName string, messages []map[string]interface{}) (string, error) {
		return s.llmClient.ChatWithModel(modelInfo, modelName, messages)
	})

	tasks := s.loadBenchmarkTasksFromDB()
	suite := benchmark.NewSuite(s.db, s.config, chatFn).WithTasks(tasks)
	suite.OnProgress = func(modelName string, taskIndex, totalTasks int, result *benchmark.TaskResult) {
		s.benchmarkOpMutex.Lock()
		defer s.benchmarkOpMutex.Unlock()
		if s.benchmarkOperation == nil {
			return
		}
		s.benchmarkOperation.CompletedTasks = taskIndex + 1
		s.benchmarkOperation.TotalTasks = totalTasks
		s.benchmarkOperation.CurrentTask = result.TaskID

		entry := logEntry{
			Model:    modelName,
			Task:     result.TaskID,
			Category: string(result.Category),
			Score:    result.Score,
			Ok:       result.Err == nil,
		}
		s.benchmarkOperation.LogEntries = append(s.benchmarkOperation.LogEntries, entry)
		if len(s.benchmarkOperation.LogEntries) > 50 {
			s.benchmarkOperation.LogEntries = s.benchmarkOperation.LogEntries[len(s.benchmarkOperation.LogEntries)-50:]
		}
	}
	if err := suite.Run(ctx, enabledModels, categoryFilter, modelFilter); err != nil {
		log.Printf("benchmark: run error: %v", err)
	}

	// Reload so in-memory ModelInfo has fresh benchmark scores.
	if err := s.loadModelsFromDatabase(); err != nil {
		log.Printf("benchmark: error reloading models after run: %v", err)
	}
}

// runBenchmarkScheduler runs benchmarks on a fixed interval until the process exits.
func (s *Server) runBenchmarkScheduler() {
	ticker := time.NewTicker(s.config.BenchmarkScheduleInterval)
	defer ticker.Stop()
	for range ticker.C {
		log.Println("benchmark: scheduled run starting")
		s.runBenchmarks(context.Background(), "", "")
	}
}

// disableEmptyTagModels reads all models from the database and disables any that have
// empty or unusable capability tags. This is a cleanup pass run after a full model refresh
// to catch models that answered the tagging request but returned no strengths.
func (s *Server) disableEmptyTagModels() {
	dbModels, err := s.db.GetAllModels()
	if err != nil {
		log.Printf("disableEmptyTagModels: error fetching models: %v", err)
		return
	}

	disabled := 0
	for name, m := range dbModels {
		source, _ := m["source"].(string)
		tags, _ := m["tags"].(string)
		enabled, _ := m["enabled"].(bool)

		// Only act on currently enabled models that lack useful tags.
		if enabled && !hasUsefulTags(tags) {
			log.Printf("Disabling %s (%s): empty capability tags", name, source)
			if err := s.db.EnableModel(name, source, false); err != nil {
				log.Printf("Error disabling %s: %v", name, err)
			} else {
				disabled++
			}
		}
	}
	if disabled > 0 {
		log.Printf("Disabled %d model(s) with empty capability tags", disabled)
	}
}
