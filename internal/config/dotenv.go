package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// dotenvPath returns the path to the .env file in the working directory.
func dotenvPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return filepath.Join(dir, ".env"), nil
}

// ReadDotEnv reads and parses the .env file. Returns an empty map if the file does not exist.
func ReadDotEnv() (map[string]string, error) {
	path, err := dotenvPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	values, err := godotenv.Read(path)
	if err != nil {
		return nil, fmt.Errorf("read .env: %w", err)
	}
	return values, nil
}

// PatchDotEnv reads the existing .env, merges the updates, and writes atomically.
func PatchDotEnv(updates map[string]string) error {
	existing, err := ReadDotEnv()
	if err != nil {
		return err
	}
	for k, v := range updates {
		existing[k] = v
	}
	return writeDotEnv(existing)
}

// RemoveDotEnvKey removes a key from the .env file.
func RemoveDotEnvKey(key string) error {
	existing, err := ReadDotEnv()
	if err != nil {
		return err
	}
	delete(existing, key)
	return writeDotEnv(existing)
}

// writeDotEnv writes all values to the .env file atomically via a temp file + rename.
func writeDotEnv(values map[string]string) error {
	path, err := dotenvPath()
	if err != nil {
		return err
	}
	content, err := godotenv.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal .env: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return fmt.Errorf("write temp .env: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp .env: %w", err)
	}
	return nil
}
