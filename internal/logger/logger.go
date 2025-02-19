package logger

import (
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
)

// LogLevel represents different levels of logging
type LogLevel int

const (
	VERBOSE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
)

var currentLevel = INFO

// SetLogLevel sets the logging level
func SetLogLevel(level LogLevel) {
	currentLevel = level
}

var warningsEncountered = []string{}
var timings = []string{}

// Logger represents a logger instance with a prefix
type Logger struct {
	prefix string
}

// GetSubLog creates a sub-logger with a specific prefix
func GetSubLog(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

func handleFormat(format string, v ...interface{}) string {
	if len(v) == 0 {
		return format
	}
	return fmt.Sprintf(format, v...)
}

func (l *Logger) createPrefix(prefix string) string {
	if currentLevel > VERBOSE {
		return ""
	}

	if l.prefix != "" {
		return fmt.Sprintf("[%s] {%s} ", prefix, l.prefix)
	}
	return fmt.Sprintf("[%s] ", prefix)
}

// Verbose logs verbose messages
func (l *Logger) Verbose(format string, v ...interface{}) {
	if currentLevel <= VERBOSE {
		prefix := l.createPrefix("V")
		log.Print(color.New(color.BgWhite, color.FgBlack).Sprint(prefix + handleFormat(format, v...)))
	}
}

// Debug logs debug messages if the level is set to DEBUG
func (l *Logger) Debug(format string, v ...interface{}) {
	if currentLevel <= DEBUG {
		prefix := l.createPrefix("D")
		log.Print(color.CyanString(prefix + handleFormat(format, v...)))
	}
}

// Info logs informational messages
func (l *Logger) Info(format string, v ...interface{}) {
	if currentLevel <= INFO {
		prefix := l.createPrefix("I")
		log.Print(color.BlueString(prefix + handleFormat(format, v...)))
	}
}

// Warn logs warning messages
func (l *Logger) Warn(format string, v ...interface{}) {
	if currentLevel <= WARN {
		prefix := l.createPrefix("W")
		warning := color.YellowString(prefix + handleFormat(format, v...))
		log.Print(warning)
		warningsEncountered = append(warningsEncountered, warning)
	}
}

// Error logs error messages
func (l *Logger) Error(format string, v ...interface{}) {
	if currentLevel <= ERROR {
		prefix := l.createPrefix("E")
		log.Print(color.RedString(prefix + handleFormat(format, v...)))
	}
}

func (l *Logger) Timing(format string, v ...interface{}) {
	if currentLevel <= INFO {
		prefix := l.createPrefix("T")
		tStr := color.MagentaString(prefix + handleFormat(format, v...))
		log.Print(tStr)
		timings = append(timings, tStr)
	}
}

func (l *Logger) TimingNoLog(format string, v ...interface{}) {
	if currentLevel <= INFO {
		prefix := l.createPrefix("T")
		tStr := color.MagentaString(prefix + handleFormat(format, v...))
		timings = append(timings, tStr)
	}
}

func (l *Logger) Prompt(format string, v ...interface{}) {
	if currentLevel <= INFO {
		prefix := l.createPrefix("?")
		fmt.Print(color.New(color.Bold, color.FgHiYellow).Sprint(prefix + handleFormat(format, v...)))
	}
}

func (l *Logger) Success(format string, v ...interface{}) {
	if currentLevel <= INFO {
		prefix := l.createPrefix("✔")
		log.Print(color.New(color.Bold, color.FgHiGreen).Sprint(prefix + handleFormat(format, v...)))
	}
}

// InitLogger initializes the logger with a default configuration
func InitLogger() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
}

// Global logger instance without a prefix
var DefaultLogger = &Logger{prefix: ""}

// Global logging functions without requiring a sub-logger
func Verbose(format string, v ...interface{})     { DefaultLogger.Verbose(format, v...) }
func Debug(format string, v ...interface{})       { DefaultLogger.Debug(format, v...) }
func Info(format string, v ...interface{})        { DefaultLogger.Info(format, v...) }
func Warn(format string, v ...interface{})        { DefaultLogger.Warn(format, v...) }
func Error(format string, v ...interface{})       { DefaultLogger.Error(format, v...) }
func Timing(format string, v ...interface{})      { DefaultLogger.Timing(format, v...) }
func TimingNoLog(format string, v ...interface{}) { DefaultLogger.TimingNoLog(format, v...) }
func Prompt(format string, v ...interface{})      { DefaultLogger.Prompt(format, v...) }
func Success(format string, v ...interface{})     { DefaultLogger.Success(format, v...) }

func TimingSummary() {
	if len(timings) == 0 {
		return
	}
	fmt.Println("")

	Info("Timing Summary:")
	for _, timing := range timings {
		Timing("* %s", timing)
	}
}

func WarningsEncountered() {
	if len(warningsEncountered) == 0 {
		return
	}
	fmt.Println("")

	Error("One or more warnings were encountered during the build process.")
	Error("For your convenience, here is a list of all warnings encountered:")
	for _, warning := range warningsEncountered {
		Warn("* %s", warning)
	}
}
