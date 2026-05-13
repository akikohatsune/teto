package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func AutoMergeDotenv() {
	examplePath := ".env.example"
	envPath := ".env"

	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		return
	}

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		fmt.Println("[INFO] .env file not found. Creating a new one from .env.example...")
		copyFile(examplePath, envPath)
		fmt.Println("[SUCCESS] .env created successfully. Please fill in your credentials.")
		return
	}

	exampleKeys := getKeysFromEnv(examplePath)
	envKeys := getKeysFromEnv(envPath)

	var missingKeys []string
	for key := range exampleKeys {
		if _, exists := envKeys[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) == 0 {
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("      SYSTEM: ENVIRONMENT SYNCHRONIZATION      ")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Detected %d missing configurations in your .env file.\n", len(missingKeys))

	f, err := os.OpenFile(envPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[WARNING] Could not open .env for appending: %v\n", err)
		return
	}
	defer f.Close()

	f.WriteString("\n\n# --- AUTO-MERGED KEYS ---\n")
	for _, key := range missingKeys {
		fmt.Printf("  + Adding: %s\n", key)
		f.WriteString(exampleKeys[key] + "\n")
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("COMPLETED: Your .env has been updated with default values.")
	fmt.Println("IMPORTANT: Please check .env and provide necessary values.")
	fmt.Println(strings.Repeat("=", 60) + "\n")
}

func getKeysFromEnv(path string) map[string]string {
	keys := make(map[string]string)
	file, err := os.Open(path)
	if err != nil {
		return keys
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(line, "=") {
			key := strings.Split(line, "=")[0]
			keys[strings.TrimSpace(key)] = line
		}
	}
	return keys
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func ClearPycache() {
	// Not needed in Go, but keeping the name if we want to clear something else
}


