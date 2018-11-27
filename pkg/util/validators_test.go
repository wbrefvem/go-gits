package util_test

import (
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1"
	"testing"
)

func TestNoWhitespaceValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName	string
		domainName  string
		want		string
	}{
		{"leading whitespace"," fake.com", "supplied value \" fake.com\" must not contain any whitespace"},
		{"trailing whitespace","fake.com ", "supplied value \"fake.com \" must not contain any whitespace"},
		{"embedded whitespace","fake .com", "supplied value \"fake .com\" must not contain any whitespace"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.want, testInputValidation(t, tt.domainName))
		})
	}
}

func testInputValidation(t *testing.T, s string) interface{} {
	valid := survey.ComposeValidators(
		util.NoWhiteSpaceValidator(),
	)
	err := valid(s)
	if err != nil {
		return err.Error()
	}
	return ""
}

