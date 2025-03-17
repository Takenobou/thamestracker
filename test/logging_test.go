package test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/scraper/vessels"
	"github.com/stretchr/testify/assert"
)

// newTestLogger creates a Zap logger that writes to a buffer.
func newTestLogger() (*zap.SugaredLogger, *bytes.Buffer) {
	buf := new(bytes.Buffer)
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, zapcore.AddSync(buf), zap.DebugLevel)
	return zap.New(core).Sugar(), buf
}

func TestLoggingVerification(t *testing.T) {
	// Save the original logger and config URL.
	originalLogger := logger.Logger
	originalURL := config.AppConfig.URLs.PortOfLondon

	// Replace with our test logger.
	testLogger, buf := newTestLogger()
	logger.Logger = testLogger
	defer func() {
		logger.Logger = originalLogger
		config.AppConfig.URLs.PortOfLondon = originalURL
	}()

	// Create a fake HTTP server that returns malformed JSON.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "malformed json")
	}))
	defer server.Close()

	// Override the PortOfLondon URL so that ScrapeVessels uses our fake server.
	config.AppConfig.URLs.PortOfLondon = server.URL

	// Call ScrapeVessels which should trigger an error log due to malformed JSON.
	_, err := vessels.ScrapeVessels("inport")
	assert.Error(t, err)

	// Verify that the logger captured the expected error message.
	logOutput := buf.String()
	assert.Contains(t, logOutput, "Error decoding API response", "expected error log message not found")
}
