package git

import (
	"errors"
	"os"
	"strings"
	"testing"

	mocks "github.com/jenkins-x/jx/pkg/gits/mocks"
	utiltests "github.com/jenkins-x/jx/pkg/tests"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/stretchr/testify/assert"
)

type FakeOrgLister struct {
	orgNames []string
	fail     bool
}

func (l FakeOrgLister) ListOrganisations() ([]Organisation, error) {
	if l.fail {
		return nil, errors.New("fail")
	}

	orgs := make([]Organisation, len(l.orgNames))
	for _, v := range l.orgNames {
		orgs = append(orgs, Organisation{Login: v})
	}
	return orgs, nil
}

func Test_getOrganizations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		testDescription string
		orgLister       OrganisationLister
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
			result := GetOrganizations(tt.orgLister, tt.userName)
			assert.Equal(t, tt.want, result)
		})
	}
}

func createGitProvider(t *testing.T, kind string, git Gitter) GitProvider {
	switch kind {
	case KindGitHub:
		gitHubProvider, err := NewGitHubProvider(server, user, git)
		assert.NoError(t, err, "should create GitHub provider without error")
		return gitHubProvider
	case KindGitlab:
		gitlabProvider, err := NewGitlabProvider(server, user, git)
		assert.NoError(t, err, "should create Gitlab provider without error")
		return gitlabProvider
	case KindGitea:
		giteaProvider, err := NewGiteaProvider(server, user, git)
		assert.NoError(t, err, "should create Gitea provider without error")
		return giteaProvider
	case KindBitBucketServer:
		bitbucketServerProvider, err := NewBitbucketServerProvider(server, user, git)
		assert.NoError(t, err, "should create Bitbucket server  provider without error")
		return bitbucketServerProvider
	case KindBitBucketCloud:
		bitbucketCloudProvider, err := NewBitbucketCloudProvider(server, user, git)
		assert.NoError(t, err, "should create Bitbucket cloud  provider without error")
		return bitbucketCloudProvider
	default:
		return nil
	}
}

func setUserAuthInEnv(kind string, username string, apiToken string) error {
	prefix := strings.ToUpper(kind)
	err := os.Setenv(prefix+"_USERNAME", username)
	if err != nil {
		return err
	}
	return os.Setenv(prefix+"_API_TOKEN", apiToken)
}

func unsetUserAuthInEnv(kind string) error {
	prefix := strings.ToUpper(kind)
	err := os.Unsetenv(prefix + "_USERNAME")
	if err != nil {
		return err
	}
	return os.Unsetenv(prefix + "_API_TOKEN")
}

func getAndCleanEnviron(kind string) (map[string]string, error) {
	prefix := strings.ToUpper(kind)
	keys := []string{
		prefix + "_USERNAME",
		prefix + "_API_TOKEN",
		"GIT_USERNAME",
		"GIT_API_TOKEN",
	}
	return util.GetAndCleanEnviron(keys)
}

func restoreEnviron(t *testing.T, environ map[string]string) {
	err := util.RestoreEnviron(environ)
	assert.NoError(t, err, "should restore the env variable")
}

func TestCreateGitProviderFromURL(t *testing.T) {
	t.Parallel()
	utiltests.SkipForWindows(t, "go-expect does not work on Windows")

	git := mocks.NewMockGitter()

	tests := []struct {
		description  string
		setup        func(t *testing.T) (*utiltests.ConsoleWrapper, chan struct{})
		cleanup      func(c *utiltests.ConsoleWrapper, donech chan struct{})
		Name         string
		providerKind string
		hostURL      string
		git          Gitter
		numUsers     int
		currUser     int
		pipelineUser int
		username     string
		apiToken     string
		batchMode    bool
		inCluster    bool
		wantError    bool
	}{
		{"create GitHub provider for one user",
			nil,
			nil,
			"GitHub",
			KindGitHub,
			"https://github.com",
			git,
			1,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create GitHub provider for multiple users",
			nil,
			nil,
			"GitHub",
			KindGitHub,
			"https://github.com",
			git,
			2,
			1,
			1,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create GitHub provider for pipline user when in cluster ",
			nil,
			nil,
			"GitHub",
			KindGitHub,
			"https://github.com",
			git,
			2,
			1,
			0,
			"test",
			"test",
			false,
			true,
			false,
		},
		{"create GitHub provider for user from environment",
			func(t *testing.T) (*utiltests.ConsoleWrapper, chan struct{}) {
				err := setUserAuthInEnv(KindGitHub, "test", "test")
				assert.NoError(t, err, "should configure the user auth in environment")
				console := utiltests.NewTerminal(t)
				donech := make(chan struct{})
				go func() {
					defer close(donech)
				}()
				return console, donech
			},
			func(c *utiltests.ConsoleWrapper, donech chan struct{}) {
				err := unsetUserAuthInEnv(KindGitHub)
				assert.NoError(t, err, "should reset the user auth in environment")
				err = c.Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"GitHub",
			KindGitHub,
			"https://github.com",
			git,
			0,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create GitHub provider in barch mode ",
			nil,
			nil,
			"GitHub",
			KindGitHub,
			"https://github.com",
			git,
			0,
			0,
			0,
			"",
			"",
			true,
			false,
			true,
		},
		{"create GitHub provider in interactive mode",
			func(t *testing.T) (*utiltests.ConsoleWrapper, chan struct{}) {
				c := utiltests.NewTerminal(t)
				assert.NotNil(t, c, "console should not be nil")
				assert.NotNil(t, c.Stdio, "term should not be nil")
				donech := make(chan struct{})
				go func() {
					defer close(donech)
					c.ExpectString("github.com user name:")
					c.SendLine("test")
					c.ExpectString("API Token:")
					c.SendLine("test")
					c.ExpectEOF()
				}()
				return c, donech
			},
			func(c *utiltests.ConsoleWrapper, donech chan struct{}) {
				err := c.Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"GitHub",
			KindGitHub,
			"https://github.com",
			git,
			0,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create Gitlab provider for one user",
			nil,
			nil,
			"Gitlab",
			KindGitlab,
			"https://gitlab.com",
			git,
			1,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create Gitlab provider for multiple users",
			nil,
			nil,
			"Gitlab",
			KindGitHub,
			"https://gitlab.com",
			git,
			2,
			1,
			1,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create Gitlab provider for user from environment",
			func(t *testing.T) (*utiltests.ConsoleWrapper, chan struct{}) {
				err := setUserAuthInEnv(KindGitlab, "test", "test")
				assert.NoError(t, err, "should configure the user auth in environment")
				c := utiltests.NewTerminal(t)
				donech := make(chan struct{})
				go func() {
					defer close(donech)
				}()
				return c, donech
			},
			func(c *utiltests.ConsoleWrapper, donech chan struct{}) {
				err := unsetUserAuthInEnv(KindGitlab)
				assert.NoError(t, err, "should reset the user auth in environment")
				err = c.Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"Gitlab",
			KindGitlab,
			"https://gitlab.com",
			git,
			0,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create Gitlab provider in barch mode ",
			nil,
			nil,
			"Gitlab",
			KindGitlab,
			"https://gitlab.com",
			git,
			0,
			0,
			0,
			"",
			"",
			true,
			false,
			true,
		},
		{"create Gitlab provider in interactive mode",
			func(t *testing.T) (*utiltests.ConsoleWrapper, chan struct{}) {
				c := utiltests.NewTerminal(t)
				assert.NotNil(t, c, "console should not be nil")
				assert.NotNil(t, c.Stdio, "term should not be nil")
				donech := make(chan struct{})
				go func() {
					defer close(donech)
					c.ExpectString("gitlab.com user name:")
					c.SendLine("test")
					c.ExpectString("API Token:")
					c.SendLine("test")
					c.ExpectEOF()
				}()
				return c, donech
			},
			func(c *utiltests.ConsoleWrapper, donech chan struct{}) {
				err := c.Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"Gitlab",
			KindGitlab,
			"https://gitlab.com",
			git,
			0,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create Gitea provider for one user",
			nil,
			nil,
			"Gitea",
			KindGitea,
			"https://gitea.com",
			git,
			1,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create Gitea provider for multiple users",
			nil,
			nil,
			"Gitea",
			KindGitea,
			"https://gitea.com",
			git,
			2,
			1,
			1,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create Gitea provider for user from environment",
			func(t *testing.T) (*utiltests.ConsoleWrapper, chan struct{}) {
				err := setUserAuthInEnv(KindGitea, "test", "test")
				assert.NoError(t, err, "should configure the user auth in environment")
				c := utiltests.NewTerminal(t)
				donech := make(chan struct{})
				go func() {
					defer close(donech)
				}()
				return c, donech
			},
			func(c *utiltests.ConsoleWrapper, donech chan struct{}) {
				err := unsetUserAuthInEnv(KindGitea)
				assert.NoError(t, err, "should reset the user auth in environment")
				err = c.Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"Gitea",
			KindGitea,
			"https://gitea.com",
			git,
			0,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create Gitea provider in barch mode ",
			nil,
			nil,
			"Gitea",
			KindGitea,
			"https://gitea.com",
			git,
			0,
			0,
			0,
			"",
			"",
			true,
			false,
			true,
		},
		{"create Gitea provider in interactive mode",
			func(t *testing.T) (*utiltests.ConsoleWrapper, chan struct{}) {
				c := utiltests.NewTerminal(t)
				assert.NotNil(t, c, "console should not be nil")
				assert.NotNil(t, c.Stdio, "term should not be nil")
				donech := make(chan struct{})
				go func() {
					defer close(donech)
					c.ExpectString("gitea.com user name:")
					c.SendLine("test")
					c.ExpectString("API Token:")
					c.SendLine("test")
					c.ExpectEOF()
				}()
				return c, donech
			},
			func(c *utiltests.ConsoleWrapper, donech chan struct{}) {
				err := c.Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"Gitea",
			KindGitea,
			"https://gitea.com",
			git,
			0,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create BitbucketServer provider for one user",
			nil,
			nil,
			"BitbucketServer",
			KindBitBucketServer,
			"https://bitbucket-server.com",
			git,
			1,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create BitbucketServer provider for multiple users",
			nil,
			nil,
			"BitbucketServer",
			KindBitBucketServer,
			"https://bitbucket-server.com",
			git,
			2,
			1,
			1,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create BitbucketServer provider for user from environment",
			func(t *testing.T) (*utiltests.ConsoleWrapper, chan struct{}) {
				err := setUserAuthInEnv(KindBitBucketServer, "test", "test")
				assert.NoError(t, err, "should configure the user auth in environment")
				c := utiltests.NewTerminal(t)
				donech := make(chan struct{})
				go func() {
					defer close(donech)
				}()
				return c, donech
			},
			func(c *utiltests.ConsoleWrapper, donech chan struct{}) {
				err := unsetUserAuthInEnv(KindBitBucketServer)
				assert.NoError(t, err, "should reset the user auth in environment")
				err = c.Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"BitbucketServer",
			KindBitBucketServer,
			"https://bitbucket-server.com",
			git,
			0,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create BitbucketServer provider in barch mode ",
			nil,
			nil,
			"BitbucketServer",
			KindBitBucketServer,
			"https://bitbucket-server.com",
			git,
			0,
			0,
			0,
			"",
			"",
			true,
			false,
			true,
		},
		{"create BitbucketServer provider in interactive mode",
			func(t *testing.T) (*utiltests.ConsoleWrapper, chan struct{}) {
				c := utiltests.NewTerminal(t)
				assert.NotNil(t, c, "console should not be nil")
				assert.NotNil(t, c.Stdio, "term should not be nil")
				donech := make(chan struct{})
				go func() {
					defer close(donech)
					c.ExpectString("bitbucket-server.com user name:")
					c.SendLine("test")
					c.ExpectString("API Token:")
					c.SendLine("test")
					c.ExpectEOF()
				}()
				return c, donech
			},
			func(c *utiltests.ConsoleWrapper, donech chan struct{}) {
				err := c.Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"BitbucketServer",
			KindBitBucketServer,
			"https://bitbucket-server.com",
			git,
			0,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create BitbucketCloud provider for one user",
			nil,
			nil,
			"BitbucketCloud",
			KindBitBucketCloud,
			"https://bitbucket.org",
			git,
			1,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create BitbucketCloud provider for multiple users",
			nil,
			nil,
			"BitbucketCloud",
			KindBitBucketCloud,
			"https://bitbucket.org",
			git,
			2,
			1,
			1,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create BitbucketCloud provider for user from environment",
			func(t *testing.T) (*utiltests.ConsoleWrapper, chan struct{}) {
				err := setUserAuthInEnv(KindBitBucketCloud, "test", "test")
				assert.NoError(t, err, "should configure the user auth in environment")
				c := utiltests.NewTerminal(t)
				donech := make(chan struct{})
				go func() {
					defer close(donech)
				}()
				return c, donech
			},
			func(c *utiltests.ConsoleWrapper, donech chan struct{}) {
				err := unsetUserAuthInEnv(KindBitBucketCloud)
				assert.NoError(t, err, "should reset the user auth in environment")
				err = c.Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"BitbucketCloud",
			KindBitBucketCloud,
			"https://bitbucket.org",
			git,
			0,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
		{"create BitbucketCloud provider in barch mode ",
			nil,
			nil,
			"BitbucketCloud",
			KindBitBucketCloud,
			"https://bitbucket.org",
			git,
			0,
			0,
			0,
			"",
			"",
			true,
			false,
			true,
		},
		{"create BitbucketCloud provider in interactive mode",
			func(t *testing.T) (*utiltests.ConsoleWrapper, chan struct{}) {
				c := utiltests.NewTerminal(t)
				assert.NotNil(t, c, "console should not be nil")
				assert.NotNil(t, c.Stdio, "term should not be nil")
				donech := make(chan struct{})
				go func() {
					defer close(donech)
					c.ExpectString("bitbucket.org user name:")
					c.SendLine("test")
					c.ExpectString("API Token:")
					c.SendLine("test")
					c.ExpectEOF()
				}()
				return c, donech
			},
			func(c *utiltests.ConsoleWrapper, donech chan struct{}) {
				err := c.Close()
				assert.NoError(t, err, "should close the tty")
				<-donech
			},
			"BitbucketCloud",
			KindBitBucketCloud,
			"https://bitbucket.org",
			git,
			0,
			0,
			0,
			"test",
			"test",
			false,
			false,
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			environ, err := getAndCleanEnviron(tc.providerKind)
			assert.NoError(t, err, "should clean the env variables")
			defer restoreEnviron(t, environ)

			var console *utiltests.ConsoleWrapper
			var donech chan struct{}
			if tc.setup != nil {
				console, donech = tc.setup(t)
			}

			var result GitProvider
			if console != nil {
				result, err = CreateProviderForURL(tc.inCluster, *authSvc, tc.providerKind, tc.hostURL, tc.git, tc.batchMode, console.In, console.Out, console.Err)
			} else {
				result, err = CreateProviderForURL(tc.inCluster, *authSvc, tc.providerKind, tc.hostURL, tc.git, tc.batchMode, nil, nil, nil)
			}
			if tc.wantError {
				assert.Error(t, err, "should fail to create provider")
				assert.Nil(t, result, "created provider should be nil")
			} else {
				assert.NoError(t, err, "should create provider without error")
				assert.NotNil(t, result, "created provider should not be nil")
				if tc.inCluster {
					want := createGitProvider(t, tc.providerKind, server, pipelineUser, tc.git)
					assert.NotNil(t, want, "expected provider should not be nil")
					assertProvider(t, want, result)
				} else {
					want := createGitProvider(t, tc.providerKind, server, currUser, tc.git)
					assert.NotNil(t, want, "expected provider should not be nil")
					assertProvider(t, want, result)
				}
			}

			if tc.cleanup != nil {
				tc.cleanup(console, donech)
			}
		})
	}
}

func assertProvider(t *testing.T, want GitProvider, result GitProvider) {
	assert.Equal(t, want.Kind(), result.Kind())
	assert.Equal(t, want.ServerURL(), result.ServerURL())
	assert.Equal(t, want.UserAuth(), result.UserAuth())
}
