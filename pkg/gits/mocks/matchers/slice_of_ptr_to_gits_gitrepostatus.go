// Code generated by pegomock. DO NOT EDIT.
package matchers

import (
	"reflect"
	"github.com/petergtz/pegomock"
	gits "github.com/jenkins-x/jx/pkg/gits"
)

func AnySliceOfPtrToGitsGitRepoStatus() []*gits.GitRepoStatus {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*([]*gits.GitRepoStatus))(nil)).Elem()))
	var nullValue []*gits.GitRepoStatus
	return nullValue
}

func EqSliceOfPtrToGitsGitRepoStatus(value []*gits.GitRepoStatus) []*gits.GitRepoStatus {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue []*gits.GitRepoStatus
	return nullValue
}
