package secrets

import (
	"github.com/joho/godotenv"
)

// LoadSecrets loads secrets from a .env file into the environment.
func LoadSecrets() error {
	return godotenv.Load()
}
