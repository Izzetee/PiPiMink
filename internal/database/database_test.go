package database

import (
	"testing"

	"PiPiMink/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Test überspringen, da er eine echte Datenbankverbindung benötigt
	t.Skip("Skipping as it requires a real database connection")

	cfg := &config.Config{
		DatabaseURL: ":memory:",
	}

	db, err := New(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, db)
}

func TestInitSchema(t *testing.T) {
	// Test überspringen, da er eine echte Datenbankverbindung benötigt
	t.Skip("Skipping as it requires a real database connection")
}

func TestSaveAndGetModels(t *testing.T) {
	// Test überspringen, da er eine echte Datenbankverbindung benötigt
	t.Skip("Skipping as it requires a real database connection")

	// Da der Test übersprungen wird, sind die Implementierungsdetails hier nicht so wichtig,
	// aber hier ist ein Beispiel, wie der Test aussehen würde:

	// db := setupTestDB(t)
	// defer db.Close()

	// Modell speichern
	// err := db.SaveModel("test-model-1", "openai", `{"capabilities":["general","math"]}`, true)
	// assert.NoError(t, err)

	// Antwort speichern
	// err = db.SaveModelResponse("test-model-1", "openai", "Model description")
	// assert.NoError(t, err)

	// Modelle abrufen
	// retrievedModels, err := db.GetAllModels()
	// assert.NoError(t, err)
	// assert.Len(t, retrievedModels, 1)

	// Inhalte überprüfen
	// model, ok := retrievedModels["test-model-1"]
	// assert.True(t, ok)
	// assert.Equal(t, "openai", model["source"])
	// assert.True(t, model["enabled"].(bool))
	// assert.Contains(t, model["tags"].(string), "general")
	// assert.Contains(t, model["tags"].(string), "math")
}

func TestEnableModel(t *testing.T) {
	// Test überspringen, da er eine echte Datenbankverbindung benötigt
	t.Skip("Skipping as it requires a real database connection")

	// db := setupTestDB(t)
	// defer db.Close()

	// Modell speichern
	// err := db.SaveModel("test-model", "openai", `{"capabilities":["general"]}`, false)
	// assert.NoError(t, err)

	// Modell aktivieren
	// err = db.EnableModel("test-model", "openai", true)
	// assert.NoError(t, err)

	// Überprüfen, ob der Status aktualisiert wurde
	// retrievedModels, err := db.GetAllModels()
	// assert.NoError(t, err)
	// model, ok := retrievedModels["test-model"]
	// assert.True(t, ok)
	// assert.True(t, model["enabled"].(bool))
}

func TestHasModels(t *testing.T) {
	// Test überspringen, da er eine echte Datenbankverbindung benötigt
	t.Skip("Skipping as it requires a real database connection")

	// db := setupTestDB(t)
	// defer db.Close()

	// Anfangs sollte die Datenbank leer sein
	// hasModels, err := db.HasModels()
	// assert.NoError(t, err)
	// assert.False(t, hasModels)

	// Ein Modell hinzufügen
	// err = db.SaveModel("test-model", "openai", `{"capabilities":["general"]}`, true)
	// assert.NoError(t, err)

	// Jetzt sollte es Modelle geben
	// hasModels, err = db.HasModels()
	// assert.NoError(t, err)
	// assert.True(t, hasModels)
}
