package auth

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	url1 = "http://dummy/"
	url2 = "http://another-jenkins/"

	userDoesNotExist = "doesNotExist"
	user1            = "someone"
	user2            = "another"
	token2v2         = "tokenV2"
)

func TestAuthConfig(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp", "jx-test-jenkins-config-")
	assertNoError(t, err)

	fileName := filepath.Join(dir, "jenkins.yaml")

	t.Logf("Using config file %s\n", fileName)

	configTest := ConfigTest{
		t: t,
	}
	configTest.svc.FileName = fileName

	config := configTest.Load()

	assert.Equal(t, 0, len(config.Servers), "Should have no servers in config but got %v", config)
	assertNoAuth(t, config, url1, userDoesNotExist)

	auth1 := UserAuth{
		Username: user1,
		ApiToken: "someToken",
	}
	config = configTest.SetUserAuth(url1, auth1)

	assert.Equal(t, 1, len(config.Servers), "Number of servers")
	assert.Equal(t, 1, len(config.Servers[0].Users), "Number of auths")
	assert.Equal(t, &auth1, config.FindUserAuth(url1, user1), "loaded auth for server %s and user %s", url1, user1)
	assert.Equal(t, &auth1, config.FindUserAuth(url1, ""), "loaded auth for server %s and no user", url1)

	auth2 := UserAuth{
		Username: user2,
		ApiToken: "anotherToken",
	}
	config = configTest.SetUserAuth(url1, auth2)

	assert.Equal(t, &auth2, config.FindUserAuth(url1, user2), "Failed to find auth for server %s and user %s", url1, user2)
	assert.Equal(t, &auth1, config.FindUserAuth(url1, user1), "loaded auth for server %s and user %s", url1, user1)
	assert.Equal(t, 1, len(config.Servers), "Number of servers")
	assert.Equal(t, 2, len(config.Servers[0].Users), "Number of auths")
	assertNoAuth(t, config, url1, userDoesNotExist)

	// lets mutate the auth2
	auth2.ApiToken = token2v2
	config = configTest.SetUserAuth(url1, auth2)

	assertNoAuth(t, config, url1, userDoesNotExist)
	assert.Equal(t, 1, len(config.Servers), "Number of servers")
	assert.Equal(t, 2, len(config.Servers[0].Users), "Number of auths")
	assert.Equal(t, &auth1, config.FindUserAuth(url1, user1), "loaded auth for server %s and user %s", url1, user1)
	assert.Equal(t, &auth2, config.FindUserAuth(url1, user2), "loaded auth for server %s and user %s", url1, user2)

	auth3 := UserAuth{
		Username: user1,
		ApiToken: "server2User1Token",
	}
	configTest.SetUserAuth(url2, auth3)

	assertNoAuth(t, config, url1, userDoesNotExist)
	assert.Equal(t, 2, len(config.Servers), "Number of servers")
	assert.Equal(t, 2, len(config.Servers[0].Users), "Number of auths for server 0")
	assert.Equal(t, 1, len(config.Servers[1].Users), "Number of auths for server 1")
	assert.Equal(t, &auth1, config.FindUserAuth(url1, user1), "loaded auth for server %s and user %s", url1, user1)
	assert.Equal(t, &auth2, config.FindUserAuth(url1, user2), "loaded auth for server %s and user %s", url1, user2)
	assert.Equal(t, &auth3, config.FindUserAuth(url2, user1), "loaded auth for server %s and user %s", url2, user1)
}

type ConfigTest struct {
	t      *testing.T
	svc    AuthConfigService
	config *AuthConfig
}

func (c *ConfigTest) Load() *AuthConfig {
	config, err := c.svc.LoadConfig()
	c.config = config
	c.AssertNoError(err)
	return c.config
}

func (c *ConfigTest) SetUserAuth(url string, auth UserAuth) *AuthConfig {
	copy := auth
	c.config.SetUserAuth(url, &copy)
	c.SaveAndReload()
	return c.config
}

func (c *ConfigTest) SaveAndReload() *AuthConfig {
	err := c.svc.SaveConfig()
	c.AssertNoError(err)
	return c.Load()
}

func (c *ConfigTest) AssertNoError(err error) {
	if err != nil {
		assert.Fail(c.t, "Should not have received an error but got: %s", err)
	}
}

func assertNoAuth(t *testing.T, config *AuthConfig, url string, user string) {
	found := config.FindUserAuth(url, user)
	if found != nil {
		assert.Fail(t, "Found auth when not expecting it for server %s and user %s", url, user)
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		assert.Fail(t, "Should not have received an error but got: %s", err)
	}
}

func TestAuthConfigGetsDefaultName(t *testing.T) {
	c := &AuthConfig{}

	expectedUrl := "https://foo.com"
	server := c.GetOrCreateServer(expectedUrl)
	assert.NotNil(t, server, "No server found!")
	assert.True(t, server.Name != "", "Should have a server name!")
	assert.Equal(t, expectedUrl, server.URL, "Server.URL")
}
