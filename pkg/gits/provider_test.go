package gits_test

import (
	"errors"
	"testing"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/stretchr/testify/assert"
)

type FakeOrgLister struct {
	orgNames []string
	fail     bool
}

func (l FakeOrgLister) ListOrganisations() ([]gits.GitOrganisation, error) {
	if l.fail {
		return nil, errors.New("fail")
	}

	orgs := make([]gits.GitOrganisation, len(l.orgNames))
	for _, v := range l.orgNames {
		orgs = append(orgs, gits.GitOrganisation{Login: v})
	}
	return orgs, nil
}

func Test_getOrganizations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		testDescription string
		orgLister       gits.OrganisationLister
		userName        string
		want            []string
	}{
		{"Should return user name when ListOrganisations() fails", FakeOrgLister{fail: true}, "testuser", []string{"testuser"}},
		{"Should return user name when organization list is empty", FakeOrgLister{orgNames: []string{}}, "testuser", []string{"testuser"}},
		{"Should include user name when only 1 organization exists", FakeOrgLister{orgNames: []string{"testorg"}}, "testuser", []string{"testorg", "testuser"}},
		{"Should include user name together with all organizations when multiple exists", FakeOrgLister{orgNames: []string{"testorg", "anotherorg"}}, "testuser", []string{"anotherorg", "testorg", "testuser"}},
	}
	for _, tt := range tests {
		t.Run(tt.testDescription, func(t *testing.T) {
			result := gits.GetOrganizations(tt.orgLister, tt.userName)
			assert.Equal(t, tt.want, result)
		})
	}
}
