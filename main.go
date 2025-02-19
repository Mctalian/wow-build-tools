/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"os"

	"github.com/McTalian/wow-build-tools/cmd"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/secrets"
	"github.com/McTalian/wow-build-tools/internal/update"
)

func main() {
	secrets.LoadSecrets()
	logger.InitLogger()
	if os.Getenv("CI") == "true" {
		logger.Verbose("Running in CI environment, no automatic updates will be performed")
	} else {
		update.ConfirmAndSelfUpdate()
	}
	cmd.Execute()
}
