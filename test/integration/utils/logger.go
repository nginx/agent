package utils

import(
	"os"
	"log"
	"testing"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

var Logger Logging = log.New(os.Stderr, "", log.LstdFlags)

// Logging defines the Logger interface
type Logging interface {
	Printf(format string, v ...interface{})
}

func TestLogger(tb testing.TB) Logging {
	tb.Helper()
	return testLogger{TB: tb}
}

type testLogger struct {
	testing.TB
}

func (t testLogger) Printf(format string, v ...interface{}) {
	t.Helper()
	t.Logf(format, v...)
}

type ComposeLoggerOption struct {
	logger Logging
}

type LoggerOption struct {
	logger Logging
}

func WithLogger(logger Logging) LoggerOption {
	return LoggerOption{
		logger: logger,
	}
}

func (t testLogger) applyToComposeStack(o *compose.ComposeLoggerOption) {
	// o.Paths = t
}