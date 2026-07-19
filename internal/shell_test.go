package internal

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureLogs(t *testing.T, fn func()) string {
	t.Helper()
	oldOutput := log.Writer()
	oldFlags := log.Flags()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(oldOutput)
		log.SetFlags(oldFlags)
	}()

	fn()
	return buf.String()
}

func TestRunShellCommandLogsLikeLegacyExecutor(t *testing.T) {
	logs := captureLogs(t, func() {
		executor := CreateExecutor(false, "/bin/bash", map[string]string{"hello": "echo hi"})
		require.NoError(t, executor("hello"))
	})

	assert.Contains(t, logs, "$ /bin/bash -c echo hi")
	assert.Contains(t, logs, "[/bin/bash -c echo hi]")
	assert.Contains(t, logs, "-------- output --------")
	assert.Contains(t, logs, "hi")
	assert.Contains(t, logs, "------------------------")
}

func TestEntityExecutorUsesSameShellLogging(t *testing.T) {
	logs := captureLogs(t, func() {
		executor := CreateEntityExecutor(false, "/bin/bash", map[string]EntityConfig{
			"volume": {Type: EntityTypeNumber, Get: "echo 42"},
		})
		_, err := executor.Execute("volume", EntityOpGet, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, logs, "$ /bin/bash -c echo 42")
	assert.Contains(t, logs, "[/bin/bash -c echo 42]")
	assert.Contains(t, logs, "42")
}
