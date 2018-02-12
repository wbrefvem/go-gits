package gits

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/util"
	"gopkg.in/AlecAivazis/survey.v1"
)

type CreateRepoData struct {
	Organisation string
	RepoName     string
	FullName     string
	PrivateRepo  bool
	User         *auth.UserAuth
	GitProvider  GitProvider
}

type GitRepositoryOptions struct {
	ServerURL string
	Username  string
	ApiToken  string
}

func (d *CreateRepoData) CreateRepository() (*GitRepository, error) {
	return d.GitProvider.CreateRepository(d.Organisation, d.RepoName, d.PrivateRepo)
}

func PickNewGitRepository(out io.Writer, batchMode bool, authConfigSvc auth.AuthConfigService, defaultRepoName string, repoOptions GitRepositoryOptions) (*CreateRepoData, error) {
	config := authConfigSvc.Config()

	var err error
	var server *auth.AuthServer
	if repoOptions.ServerURL != "" {
		server = config.GetOrCreateServer(repoOptions.ServerURL)
	} else {
		server, err = config.PickServer("Which git provider?")
		if err != nil {
			return nil, err
		}
	}
	fmt.Fprintf(out, "Using git provider %s\n", util.ColorInfo(server.Description()))
	url := server.URL
	var userAuth *auth.UserAuth
	if repoOptions.Username != "" {
		userAuth = config.GetOrCreateUserAuth(url, repoOptions.Username)
	} else {
		if batchMode {
			if len(server.Users) == 0 {
				return nil, fmt.Errorf("Server %s has no user auths defined!", url)
			}
			var ua *auth.UserAuth
			if server.CurrentUser != "" {
				ua = config.FindUserAuth(url, server.CurrentUser)
			}
			if ua == nil {
				ua = server.Users[0]
			}
			userAuth = ua
		} else {
			userAuth, err = config.PickServerUserAuth(server, "git user name?")
			if err != nil {
				return nil, err
			}
		}
	}
	if userAuth.IsInvalid() && repoOptions.ApiToken != "" {
		userAuth.ApiToken = repoOptions.ApiToken
	}
	if userAuth.IsInvalid() {
		PrintCreateRepositoryGenerateAccessToken(server, out)

		// TODO could we guess this based on the users ~/.git for github?
		defaultUserName := ""
		err = config.EditUserAuth(server.Label(), userAuth, defaultUserName, true, batchMode)
		if err != nil {
			return nil, err
		}

		// TODO lets verify the auth works

		err = authConfigSvc.SaveUserAuth(url, userAuth)
		if err != nil {
			return nil, fmt.Errorf("Failed to store git auth configuration %s", err)
		}
		if userAuth.IsInvalid() {
			return nil, fmt.Errorf("You did not properly define the user authentication!")
		}
	}

	gitUsername := userAuth.Username
	fmt.Fprintf(out, "\n\nAbout to create a repository on server %s with user %s\n", util.ColorInfo(url), util.ColorInfo(gitUsername))

	provider, err := CreateProvider(server, userAuth)
	if err != nil {
		return nil, err
	}
	org, err := PickOrganisation(provider, gitUsername)
	if err != nil {
		return nil, err
	}
	owner := org
	if org == "" {
		owner = gitUsername
	}
	repoName := ""
	if batchMode {
		repoName = defaultRepoName
		if repoName == "" {
			repoName = "dummy"
		}
	} else {
		prompt := &survey.Input{
			Message: "Enter the new repository name: ",
			Default: defaultRepoName,
		}
		validator := func(val interface{}) error {
			str, ok := val.(string)
			if !ok {
				return fmt.Errorf("Expected string value!")
			}
			if strings.TrimSpace(str) == "" {
				return fmt.Errorf("Repository name is required")
			}
			return provider.ValidateRepositoryName(owner, str)
		}
		err = survey.AskOne(prompt, &repoName, validator)
		if err != nil {
			return nil, err
		}
		if repoName == "" {
			return nil, fmt.Errorf("No repository name specified!")
		}
	}
	fullName := GitRepoName(org, repoName)
	fmt.Fprintf(out, "\n\nCreating repository %s\n", util.ColorInfo(fullName))
	privateRepo := false

	return &CreateRepoData{
		Organisation: org,
		RepoName:     repoName,
		FullName:     fullName,
		PrivateRepo:  privateRepo,
		User:         userAuth,
		GitProvider:  provider,
	}, err
}
