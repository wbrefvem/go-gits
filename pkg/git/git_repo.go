package git

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/auth"
	survey "gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

type CreateRepoData struct {
	Organisation string
	RepoName     string
	FullName     string
	PrivateRepo  bool
	User         *auth.UserAuth
	Provider  Provider
}

type GitRepositoryOptions struct {
	ServerURL string
	Username  string
	ApiToken  string
	Owner     string
	RepoName  string
	Private   bool
}

// GetRepository returns the repository if it already exists
func (d *CreateRepoData) GetRepository() (*Repository, error) {
	return d.Provider.GetRepository(d.Organisation, d.RepoName)
}

// CreateRepository creates the repository - failing if it already exists
func (d *CreateRepoData) CreateRepository() (*Repository, error) {
	return d.Provider.CreateRepository(d.Organisation, d.RepoName, d.PrivateRepo)
}

func GetRepoName(batchMode, allowExistingRepo bool, provider Provider, defaultRepoName, owner string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (string, error) {
	surveyOpts := survey.WithStdio(in, out, errOut)
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
				return fmt.Errorf("Expected string value")
			}
			if strings.TrimSpace(str) == "" {
				return fmt.Errorf("Repository name is required")
			}
			if allowExistingRepo {
				return nil
			}
			return provider.ValidateRepositoryName(owner, str)
		}
		err := survey.AskOne(prompt, &repoName, validator, surveyOpts)
		if err != nil {
			return "", err
		}
		if repoName == "" {
			return "", fmt.Errorf("No repository name specified")
		}
	}
	return repoName, nil
}

func GetOwner(batchMode bool, provider Provider, gitUsername string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (string, error) {
	owner := ""
	if batchMode {
		owner = gitUsername
	} else {
		org, err := PickOrganisation(provider, gitUsername, in, out, errOut)
		if err != nil {
			return "", err
		}
		owner = org
		if org == "" {
			owner = gitUsername
		}
	}
	return owner, nil
}
