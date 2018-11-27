package system

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetOsVersionReturnsNoError(t *testing.T) {
	t.Parallel()
	ver, err := GetOsVersion()
	assert.NoError(t, err)
	assert.NotNil(t, ver)
	assert.NotEmpty(t, ver)
	//assert.Equal(t, "Windows 10 Pro 1803 build 17134", ver)
}
