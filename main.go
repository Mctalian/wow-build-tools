/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/McTalian/wow-build-tools/cmd"
	"github.com/McTalian/wow-build-tools/internal/logger"
	"github.com/McTalian/wow-build-tools/internal/update"
)

func main() {
	logger.InitLogger()
	update.ConfirmAndSelfUpdate()
	cmd.Execute()
}
