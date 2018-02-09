package gits

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/auth"
	"gopkg.in/AlecAivazis/survey.v1"
	"sort"
	"strings"
	"time"
)

type GitProvider interface {
	ListOrganisations() ([]GitOrganisation, error)

	ListRepositories(org string) ([]*GitRepository, error)

	CreateRepository(org string, name string, private bool) (*GitRepository, error)

	DeleteRepository(org string, name string) error

	ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error)

	RenameRepository(org string, name string, newName string) (*GitRepository, error)

	ValidateRepositoryName(org string, name string) error

	CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error)

	UpdatePullRequestStatus(pr *GitPullRequest) error

	CreateWebHook(data *GitWebHookArguments) error

	IsGitHub() bool

	// returns the path relative to the Jenkins URL to trigger webhooks on this kind of repository
	//

	// e.g. for GitHub its /github-webhook/
	// other examples include:
	//
	// * gitlab: /gitlab/notify_commit
	// https://github.com/elvanja/jenkins-gitlab-hook-plugin#notify-commit-hook
	//
	// * git plugin
	// /git/notifyCommit?url=
	// http://kohsuke.org/2011/12/01/polling-must-die-triggering-jenkins-builds-from-a-git-hook/
	//
	// * generic webhook
	// /generic-webhook-trigger/invoke?token=abc123
	// https://wiki.jenkins.io/display/JENKINS/Generic+Webhook+Trigger+Plugin

	JenkinsWebHookPath(gitURL string, secret string) string

	Label() string
}

type GitOrganisation struct {
	Login string
}

type GitRepository struct {
	Name             string
	AllowMergeCommit bool
	HTMLURL          string
	CloneURL         string
	SSHURL           string
}

type GitPullRequest struct {
	URL         string
	Owner       string
	Repo        string
	Number      *int
	Mergeable   *bool
	Merged      *bool
	State       *string
	StatusesURL *string
	IssueURL    *string
	DiffURL     *string
	ClosedAt    *time.Time
	MergedAt    *time.Time
}

type GitPullRequestArguments struct {
	Owner string
	Repo  string
	Title string
	Body  string
	Head  string
	Base  string
}

type GitWebHookArguments struct {
	Owner  string
	Repo   string
	URL    string
	Secret string
}

// IsClosed returns true if the PullRequest has been closed
func (pr *GitPullRequest) IsClosed() bool {
	return pr.ClosedAt != nil
}

func CreateProvider(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	switch server.Kind {
	case "gitea":
		return NewGiteaProvider(server, user)
	default:
		return NewGitHubProvider(server, user)
	}
}

func ProviderAccessTokenURL(kind string, url string) string {
	switch kind {
	case "gitea":
		return GiteaAccessTokenURL(url)
	default:
		return GitHubAccessTokenURL(url)
	}
}

// PickOrganisation picks an organisations login if there is one available
func PickOrganisation(provider GitProvider, userName string) (string, error) {
	answer := ""
	orgs, err := provider.ListOrganisations()
	if err != nil {
		return answer, err
	}
	if len(orgs) == 0 {
		return answer, nil
	}
	if len(orgs) == 1 {
		return orgs[0].Login, nil
	}
	orgNames := []string{userName}
	for _, o := range orgs {
		name := o.Login
		if name != "" {
			orgNames = append(orgNames, name)
		}
	}
	sort.Strings(orgNames)
	orgName := ""
	prompt := &survey.Select{
		Message: "Which organisation do you want to use?",
		Options: orgNames,
		Default: userName,
	}
	err = survey.AskOne(prompt, &orgName, nil)
	if err != nil {
		return "", err
	}
	if orgName == userName {
		return "", nil
	}
	return orgName, nil
}

func PickRepositories(provider GitProvider, owner string, message string, selectAll bool, filter string) ([]*GitRepository, error) {
	answer := []*GitRepository{}
	repos, err := provider.ListRepositories(owner)
	if err != nil {
		return answer, err
	}

	repoMap := map[string]*GitRepository{}
	allRepoNames := []string{}
	for _, repo := range repos {
		n := repo.Name
		if n != "" && (filter == "" || strings.Contains(n, filter)) {
			allRepoNames = append(allRepoNames, n)
			repoMap[n] = repo
		}
	}
	if len(allRepoNames) == 0 {
		return answer, fmt.Errorf("No matching repositories could be found!")
	}
	sort.Strings(allRepoNames)

	prompt := &survey.MultiSelect{
		Message: message,
		Options: allRepoNames,
	}
	if selectAll {
		prompt.Default = allRepoNames
	}
	repoNames := []string{}
	err = survey.AskOne(prompt, &repoNames, nil)

	for _, n := range repoNames {
		repo := repoMap[n]
		if repo != nil {
			answer = append(answer, repo)
		}
	}
	return answer, err
}

func (i *GitRepositoryInfo) PickOrCreateProvider(authConfigSvc auth.AuthConfigService, message string) (GitProvider, error) {
	config := authConfigSvc.Config()
	server := config.GetOrCreateServer(i.Host)
	userAuth, err := config.PickServerUserAuth(server, message)
	if err != nil {
		return nil, err
	}
	return i.CreateProviderForUser(server, userAuth)
}

func (i *GitRepositoryInfo) CreateProviderForUser(server *auth.AuthServer, user *auth.UserAuth) (GitProvider, error) {
	if i.Host == GitHubHost {
		return NewGitHubProvider(server, user)
	}
	return nil, fmt.Errorf("Git provider not supported for host %s", i.Host)
}
