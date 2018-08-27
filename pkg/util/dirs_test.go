package util_test

import (
	"errors"
	"testing"

	"github.com/jenkins-x/jx/pkg/util"
	mocks "github.com/jenkins-x/jx/pkg/util/mocks"
	. "github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"
)

func TestJXBinaryLocationSuccess(t *testing.T) {
	t.Parallel()
	commandInterface := mocks.NewMockCommandInterface()
	When(commandInterface.RunWithoutRetry()).ThenReturn("/test/something/bin/jx", nil)

	res, err := util.JXBinaryLocation(commandInterface)
	assert.Equal(t, "/test/something/bin", res)
	assert.NoError(t, err, "Should not error")
}

func TestJXBinaryLocationFailure(t *testing.T) {
	t.Parallel()
	commandInterface := mocks.NewMockCommandInterface()
	When(commandInterface.RunWithoutRetry()).ThenReturn("", errors.New("Kaboom"))

	res, err := util.JXBinaryLocation(commandInterface)
	assert.Equal(t, "", res)
	assert.Error(t, err, "Should error")
}
