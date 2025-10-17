package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// Config holds logger configuration
type Config struct {
	Level      LogLevel
	OutputFile string        // Path to log file (empty = stdout only)
	MaxSize    int64         // Max size in bytes before rotation (default: 10MB)
	MaxBackups int           // Number of old log files to keep (default: 3)
	JSONFormat bool          // Use JSON format (default: false for debug, true for production)
	AddSource  bool          // Add source file and line number (default: true in debug)
}

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	slog     *slog.Logger
	config   Config
	file     *os.File
	mu       sync.Mutex
	debugMode bool
}

var (
	globalLogger *Logger
	once         sync.Once
)

// Initialize creates and configures the global logger
// This must be called before any logging operations
func Initialize(config Config) error {
	var initErr error
	once.Do(func() {
		logger, err := NewLogger(config)
		if err != nil {
			initErr = fmt.Errorf("failed to initialize logger: %w", err)
			return
		}
		globalLogger = logger
	})
	return initErr
}

// NewLogger creates a new logger instance with the given configuration
func NewLogger(config Config) (*Logger, error) {
	// Set defaults
	if config.MaxSize == 0 {
		config.MaxSize = 10 * 1024 * 1024 // 10MB
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = 3
	}

	logger := &Logger{
		config:    config,
		debugMode: config.Level == DEBUG,
	}

	// Setup output writers
	var writers []io.Writer
	writers = append(writers, os.Stdout) // Always write to stdout

	// Add file output if specified
	if config.OutputFile != "" {
		// Ensure directory exists
		dir := filepath.Dir(config.OutputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
		}

		// Check if rotation needed
		if err := logger.rotateIfNeeded(); err != nil {
			return nil, fmt.Errorf("failed to rotate logs: %w", err)
		}

		// Open log file
		file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", config.OutputFile, err)
		}
		logger.file = file
		writers = append(writers, file)
	}

	// Create multi-writer
	multiWriter := io.MultiWriter(writers...)

	// Configure slog handler
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     logger.toSlogLevel(config.Level),
		AddSource: config.AddSource,
	}

	if config.JSONFormat {
		handler = slog.NewJSONHandler(multiWriter, opts)
	} else {
		handler = slog.NewTextHandler(multiWriter, opts)
	}

	logger.slog = slog.New(handler)
	return logger, nil
}

// rotateIfNeeded checks if log file needs rotation and performs it
func (l *Logger) rotateIfNeeded() error {
	if l.config.OutputFile == "" {
		return nil
	}

	info, err := os.Stat(l.config.OutputFile)
	if os.IsNotExist(err) {
		return nil // File doesn't exist yet
	}
	if err != nil {
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	if info.Size() < l.config.MaxSize {
		return nil // No rotation needed
	}

	// Close existing file if open
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}

	// Rotate existing backup files
	for i := l.config.MaxBackups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", l.config.OutputFile, i)
		newPath := fmt.Sprintf("%s.%d", l.config.OutputFile, i+1)
		if _, err := os.Stat(oldPath); err == nil {
			os.Rename(oldPath, newPath) // Ignore error, file might not exist
		}
	}

	// Rotate current file to .1
	backupPath := fmt.Sprintf("%s.1", l.config.OutputFile)
	if err := os.Rename(l.config.OutputFile, backupPath); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	return nil
}

// toSlogLevel converts our LogLevel to slog.Level
func (l *Logger) toSlogLevel(level LogLevel) slog.Level {
	switch level {
	case DEBUG:
		return slog.LevelDebug
	case INFO:
		return slog.LevelInfo
	case WARN:
		return slog.LevelWarn
	case ERROR:
		return slog.LevelError
	case FATAL:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Debug logs a debug message (only in debug mode)
func (l *Logger) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

// Fatal logs an error message and exits the program
func (l *Logger) Fatal(msg string, args ...any) {
	l.slog.Error(msg, args...)
	l.Close()
	os.Exit(1)
}

// With returns a new logger with additional context
func (l *Logger) With(args ...any) *Logger {
	newLogger := *l
	newLogger.slog = l.slog.With(args...)
	return &newLogger
}

// Close closes the log file if one is open
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// Global logging functions for convenience

// Debug logs a debug message using the global logger
func Debug(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Debug(msg, args...)
	} else {
		slog.Debug(msg, args...)
	}
}

// Info logs an info message using the global logger
func Info(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Info(msg, args...)
	} else {
		slog.Info(msg, args...)
	}
}

// Warn logs a warning message using the global logger
func Warn(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Warn(msg, args...)
	} else {
		slog.Warn(msg, args...)
	}
}

// Error logs an error message using the global logger
func Error(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Error(msg, args...)
	} else {
		slog.Error(msg, args...)
	}
}

// Fatal logs an error message and exits the program using the global logger
func Fatal(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Fatal(msg, args...)
	} else {
		slog.Error(msg, args...)
		os.Exit(1)
	}
}

// With returns a new logger with additional context using the global logger
func With(args ...any) *Logger {
	if globalLogger != nil {
		return globalLogger.With(args...)
	}
	return nil
}

// Close closes the global logger
func Close() error {
	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}

// IsDebugEnabled returns true if debug logging is enabled
func IsDebugEnabled() bool {
	if globalLogger != nil {
		return globalLogger.debugMode
	}
	return false
}

// GetLogFilePath returns the current log file path
func GetLogFilePath() string {
	if globalLogger != nil {
		return globalLogger.config.OutputFile
	}
	return ""
}

// LogFileInfo returns information about the current log file
func LogFileInfo() (path string, size int64, err error) {
	if globalLogger == nil || globalLogger.config.OutputFile == "" {
		return "", 0, fmt.Errorf("no log file configured")
	}

	path = globalLogger.config.OutputFile
	info, err := os.Stat(path)
	if err != nil {
		return path, 0, fmt.Errorf("failed to stat log file: %w", err)
	}

	return path, info.Size(), nil
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig(debugMode bool) Config {
	level := INFO
	if debugMode {
		level = DEBUG
	}

	logDir := "logs"
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile := filepath.Join(logDir, fmt.Sprintf("crisk_%s.log", timestamp))

	return Config{
		Level:      level,
		OutputFile: logFile,
		MaxSize:    10 * 1024 * 1024, // 10MB
		MaxBackups: 3,
		JSONFormat: !debugMode, // Human-readable in debug, JSON in production
		AddSource:  debugMode,  // Add source location in debug mode
	}
}

// DebugConfig returns a configuration optimized for debugging
func DebugConfig() Config {
	return Config{
		Level:      DEBUG,
		OutputFile: "", // stdout only for debugging
		JSONFormat: false,
		AddSource:  true,
	}
}

// ProductionConfig returns a configuration optimized for production
func ProductionConfig(logFile string) Config {
	return Config{
		Level:      INFO,
		OutputFile: logFile,
		MaxSize:    50 * 1024 * 1024, // 50MB
		MaxBackups: 10,
		JSONFormat: true,
		AddSource:  false,
	}
}
