// Code generated by pegomock. DO NOT EDIT.
package matchers

import (
	"reflect"
	"github.com/petergtz/pegomock"
	auth "github.com/jenkins-x/jx/pkg/auth"
)

func AnyPtrToAuthUserAuth() *auth.UserAuth {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*(*auth.UserAuth))(nil)).Elem()))
	var nullValue *auth.UserAuth
	return nullValue
}

func EqPtrToAuthUserAuth(value *auth.UserAuth) *auth.UserAuth {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue *auth.UserAuth
	return nullValue
}
