package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gagliardetto/solana-go"
	"gopkg.in/yaml.v3"
)

// Config represents the configuration for the mock server
type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Validator struct {
		IdentityKeypair string `yaml:"identity_keypair"`
		RunningVersion  string `yaml:"running_version"`
	} `yaml:"validator"`
	Health struct {
		StatusCode   int    `yaml:"status_code"`
		ResponseBody string `yaml:"response_body"`
	} `yaml:"health"`
}

var config Config

func main() {
	// Load configuration
	configFile := "config.yaml"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	if err := loadConfig(configFile); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set up routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", rpcHandler)

	port := config.Server.Port
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	log.Printf("Mock server starting on port %s", port)
	log.Printf("Validator identity keypair: %s", config.Validator.IdentityKeypair)
	log.Printf("Validator version: %s", config.Validator.RunningVersion)
	log.Printf("Health endpoint: %d - %s", config.Health.StatusCode, config.Health.ResponseBody)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func loadConfig(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

func loadKeypair(keypairPath string) (solana.PublicKey, error) {
	// Make path absolute if it's relative
	if !filepath.IsAbs(keypairPath) {
		// Get the directory of the current executable
		execDir, _ := os.Getwd()
		keypairPath = filepath.Join(execDir, keypairPath)
	}

	// Load the keypair using Solana SDK
	keypair, err := solana.PrivateKeyFromSolanaKeygenFile(keypairPath)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("failed to load keypair: %w", err)
	}

	return keypair.PublicKey(), nil
}

func getIdentityPubkey() string {
	publicKey, err := loadKeypair(config.Validator.IdentityKeypair)
	if err != nil {
		log.Fatalf("Failed to load keypair: %v", err)
	}

	// Convert Solana public key to base58 string
	encoded := publicKey.String()
	return encoded
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(config.Health.StatusCode)
	w.Write([]byte(config.Health.ResponseBody))
}

func rpcHandler(w http.ResponseWriter, r *http.Request) {
	// Parse JSON-RPC request
	var request struct {
		ID     int    `json:"id"`
		Method string `json:"method"`
		Params []any  `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Handle different RPC methods
	var response interface{}

	switch request.Method {
	case "getIdentity":
		response = map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      request.ID,
			"result": map[string]interface{}{
				"identity": getIdentityPubkey(),
			},
		}

	case "getHealth":
		response = map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      request.ID,
			"result":  config.Health.ResponseBody,
		}

	case "getVersion":
		response = map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      request.ID,
			"result": map[string]interface{}{
				"solana-core": config.Validator.RunningVersion,
			},
		}

	default:
		response = map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      request.ID,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
