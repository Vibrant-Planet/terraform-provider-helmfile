package helmfile

import (
	"bytes"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// OutputCapture captures log output from helmfile operations
type OutputCapture struct {
	buffer *bytes.Buffer
	mutex  sync.Mutex
}

// NewOutputCapture creates a new output capture
func NewOutputCapture() *OutputCapture {
	return &OutputCapture{
		buffer: &bytes.Buffer{},
	}
}

// Write implements io.Writer
func (o *OutputCapture) Write(p []byte) (n int, err error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return o.buffer.Write(p)
}

// String returns the captured output
func (o *OutputCapture) String() string {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return o.buffer.String()
}

// Reset clears the captured output
func (o *OutputCapture) Reset() {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.buffer.Reset()
}

// CreateCaptureLogger creates a zap logger that captures output
func CreateCaptureLogger(capture *OutputCapture) *zap.SugaredLogger {
	// Create encoder config for plain text output
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create core that writes to our capture buffer
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(capture),
		zapcore.DebugLevel, // Capture all levels
	)

	// Create logger
	logger := zap.New(core)
	return logger.Sugar()
}
