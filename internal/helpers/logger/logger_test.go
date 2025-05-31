package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerInit(t *testing.T) {
	InitLogger()
	assert.NotNil(t, Logger)
}
