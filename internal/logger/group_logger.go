package logger

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fatih/color"
)

// BufferedLog holds a single log entry that you can buffer.
type BufferedLog struct {
	Level  LogLevel
	Format string
	Args   []interface{}
}

// LogGroup represents a group of buffered log entries with a header.
type LogGroup struct {
	timeCreated time.Time
	Header      string
	Buffer      []BufferedLog
	parent      *Logger
}

// NewLogGroup creates a new LogGroup with the specified header.
func NewLogGroup(header string, args ...*Logger) *LogGroup {
	parent := DefaultLogger
	if len(args) > 0 {
		parent = args[0]
	}
	return &LogGroup{
		timeCreated: time.Now(),
		Header:      header,
		Buffer:      make([]BufferedLog, 0),
		parent:      parent,
	}
}

func (lg *LogGroup) Verbose(format string, args ...interface{}) {
	lg.add(VERBOSE, format, args...)
}

func (lg *LogGroup) Debug(format string, args ...interface{}) {
	lg.add(DEBUG, format, args...)
}

func (lg *LogGroup) Info(format string, args ...interface{}) {
	lg.add(INFO, format, args...)
}

func (lg *LogGroup) Warn(format string, args ...interface{}) {
	lg.add(WARN, format, args...)
}

func (lg *LogGroup) Error(format string, args ...interface{}) {
	lg.add(ERROR, format, args...)
}

// add appends a new buffered log entry to the group.
func (lg *LogGroup) add(level LogLevel, format string, args ...interface{}) {
	lg.Buffer = append(lg.Buffer, BufferedLog{
		Level:  level,
		Format: format,
		Args:   args,
	})
	if level == WARN {
		lg.parent.warningsEncountered = append(lg.parent.warningsEncountered, fmt.Sprintf(format, args...))
	}
}

// colorizeMessage returns a colored version of the message based on its log level.
func colorizeMessage(level LogLevel, message string) string {
	switch level {
	case VERBOSE:
		// Use white background for verbose messages.
		return color.New(color.BgWhite, color.FgBlack).Sprint(message)
	case DEBUG:
		// Use cyan for debug messages.
		return color.CyanString(message)
	case INFO:
		return color.BlueString(message)
	case WARN:
		return color.YellowString(message)
	case ERROR:
		return color.RedString(message)
	default:
		return message
	}
}

// Flush builds the group log output and prints it in one atomic call.
// It filters out any entries that are below the current log level.
func (lg *LogGroup) Flush(writeToTiming ...bool) {
	var sb strings.Builder
	withTiming := false
	if len(writeToTiming) > 0 {
		withTiming = writeToTiming[0]
	}
	// Print the header.
	// You might choose to colorize or style it as needed.
	headerStr := color.GreenString(lg.Header)

	if lg.parent.level == WARN {
		sb.WriteString(headerStr)
		sb.WriteString(color.MagentaString(fmt.Sprintf(" took %s", time.Since(lg.timeCreated))))
		log.Print(sb.String())
		return
	}

	sb.WriteString("\n")
	sb.WriteString(headerStr)
	sb.WriteString("\n")

	// Iterate through the buffered entries.
	for _, entry := range lg.Buffer {
		// Check if the entry should be printed based on current log level.
		if lg.parent.level <= entry.Level {
			// Format the message.
			msg := fmt.Sprintf(entry.Format, entry.Args...)
			// Apply level-specific color.
			coloredMsg := colorizeMessage(entry.Level, msg)
			// Indent the message (e.g. 4 spaces).
			sb.WriteString("    ")
			sb.WriteString(coloredMsg)
			sb.WriteString("\n")
		}
	}

	sb.WriteString(color.MagentaString(fmt.Sprintf("🏁  %s took %s", lg.Header, time.Since(lg.timeCreated))))

	// Print the entire group in one atomic call.
	log.Print(sb.String())
	if withTiming {
		TimingNoLog("🏁  %s took %s", lg.Header, time.Since(lg.timeCreated))
	}
}
