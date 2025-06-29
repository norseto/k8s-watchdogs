package version

import (
	"bytes"
	"testing"

	watchdogs "github.com/norseto/k8s-watchdogs"
	"github.com/stretchr/testify/assert"
)

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()
	assert.Equal(t, "version", cmd.Use)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	err := cmd.Execute()
	assert.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, watchdogs.RELEASE_VERSION)
}
