package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ModelCollectionSuite is a test suite for the ModelCollection
type ModelCollectionSuite struct {
	suite.Suite
	collection *ModelCollection
}

// SetupTest is called before each test
func (s *ModelCollectionSuite) SetupTest() {
	s.collection = NewModelCollection()
}

// TestNewModelCollection tests the constructor
func (s *ModelCollectionSuite) TestNewModelCollection() {
	assert.NotNil(s.T(), s.collection)
	assert.Empty(s.T(), s.collection.Models)
}

// TestAddModel tests adding a model to the collection
func (s *ModelCollectionSuite) TestAddModel() {
	// Test-Modell hinzufügen
	modelInfo := ModelInfo{
		Source:    "openai",
		Tags:      `{"capabilities":["general","coding"]}`,
		Enabled:   true,
		Response:  "Model description",
		UpdatedAt: "2023-05-01",
	}

	s.collection.AddModel("test-model", modelInfo)

	// Überprüfen, ob das Modell hinzugefügt wurde
	assert.Len(s.T(), s.collection.Models, 1)
	assert.Equal(s.T(), modelInfo, s.collection.Models["test-model"])
}

// TestGetModel tests retrieving models from the collection
func (s *ModelCollectionSuite) TestGetModel() {
	// Zwei Test-Modelle hinzufügen
	model1 := ModelInfo{
		Source:    "openai",
		Tags:      `{"capabilities":["general"]}`,
		Enabled:   true,
		Response:  "Model 1 description",
		UpdatedAt: "2023-05-01",
	}

	model2 := ModelInfo{
		Source:    "local",
		Tags:      `{"capabilities":["coding"]}`,
		Enabled:   false,
		Response:  "Model 2 description",
		UpdatedAt: "2023-05-02",
	}

	s.collection.AddModel("model1", model1)
	s.collection.AddModel("model2", model2)

	// Modell abrufen
	foundModel, exists := s.collection.GetModel("model1")
	assert.True(s.T(), exists)
	assert.Equal(s.T(), model1, foundModel)

	// Nicht existierendes Modell
	_, exists = s.collection.GetModel("non-existent")
	assert.False(s.T(), exists)
}

// TestGetEnabledModels tests filtering for enabled models
func (s *ModelCollectionSuite) TestGetEnabledModels() {
	// Modelle mit verschiedenen Enabled-Status hinzufügen
	s.collection.AddModel("model1", ModelInfo{Enabled: true})
	s.collection.AddModel("model2", ModelInfo{Enabled: false})
	s.collection.AddModel("model3", ModelInfo{Enabled: true})
	s.collection.AddModel("model4", ModelInfo{Enabled: false})

	enabledModels := s.collection.GetEnabledModels()

	assert.Len(s.T(), enabledModels, 2)
	_, exists := enabledModels["model1"]
	assert.True(s.T(), exists)
	_, exists = enabledModels["model3"]
	assert.True(s.T(), exists)
}

// TestFromDatabaseMap tests loading models from a database map
func (s *ModelCollectionSuite) TestFromDatabaseMap() {
	// Erstellen einer "Datenbank-Map"
	dbModels := map[string]map[string]interface{}{
		"model1": {
			"source":     "openai",
			"tags":       `{"capabilities":["general"]}`,
			"response":   "Model 1 response",
			"enabled":    true,
			"updated_at": "2023-05-01",
		},
		"model2": {
			"source":     "local",
			"tags":       `{"capabilities":["coding"]}`,
			"response":   "Model 2 response",
			"enabled":    false,
			"updated_at": "2023-05-02",
		},
	}

	// Map in die Modellsammlung laden
	s.collection.FromDatabaseMap(dbModels)

	// Überprüfen, ob die Modelle korrekt geladen wurden
	assert.Len(s.T(), s.collection.Models, 2)

	model1, exists := s.collection.GetModel("model1")
	assert.True(s.T(), exists)
	assert.Equal(s.T(), "openai", model1.Source)
	assert.Equal(s.T(), `{"capabilities":["general"]}`, model1.Tags)
	assert.True(s.T(), model1.Enabled)

	model2, exists := s.collection.GetModel("model2")
	assert.True(s.T(), exists)
	assert.Equal(s.T(), "local", model2.Source)
	assert.Equal(s.T(), `{"capabilities":["coding"]}`, model2.Tags)
	assert.False(s.T(), model2.Enabled)
}

// Run the test suite
func TestModelCollectionSuite(t *testing.T) {
	suite.Run(t, new(ModelCollectionSuite))
}

// Standalone tests that don't need the suite

func TestValidateJSONTags(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid JSON",
			input:    `{"capabilities":["math","coding"]}`,
			expected: `{"capabilities":["math","coding"]}`,
		},
		{
			name:     "Empty String",
			input:    "",
			expected: "{}",
		},
		{
			name:     "Invalid JSON",
			input:    `{"capabilities":["math","coding"`,
			expected: "{}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateJSONTags(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
