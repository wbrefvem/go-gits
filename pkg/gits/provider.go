package gits

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

type GitOrganisation struct {
	Login string
}

type GitRepository struct {
	Name             string
	AllowMergeCommit bool
	HTMLURL          string
	CloneURL         string
	SSHURL           string
	Language         string
	Fork             bool
	Stars            int
}

type GitPullRequest struct {
	URL            string
	Author         *GitUser
	Owner          string
	Repo           string
	Number         *int
	Mergeable      *bool
	Merged         *bool
	HeadRef        *string
	State          *string
	StatusesURL    *string
	IssueURL       *string
	DiffURL        *string
	MergeCommitSHA *string
	ClosedAt       *time.Time
	MergedAt       *time.Time
	LastCommitSha  string
	Title          string
	Body           string
}

type GitCommit struct {
	SHA       string
	Message   string
	Author    *GitUser
	URL       string
	Branch    string
	Committer *GitUser
}

type GitIssue struct {
	URL           string
	Owner         string
	Repo          string
	Number        *int
	Key           string
	Title         string
	Body          string
	State         *string
	Labels        []GitLabel
	StatusesURL   *string
	IssueURL      *string
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
	ClosedAt      *time.Time
	IsPullRequest bool
	User          *GitUser
	ClosedBy      *GitUser
	Assignees     []GitUser
}

type GitUser struct {
	URL       string
	Login     string
	Name      string
	Email     string
	AvatarURL string
}

type GitRelease struct {
	Name          string
	TagName       string
	Body          string
	URL           string
	HTMLURL       string
	DownloadCount int
}

type GitLabel struct {
	URL   string
	Name  string
	Color string
}

type GitRepoStatus struct {
	ID      string
	Context string
	URL     string

	// State is the current state of the repository. Possible values are:
	// pending, success, error, or failure.
	State string `json:"state,omitempty"`

	// TargetURL is the URL of the page representing this status
	TargetURL string `json:"target_url,omitempty"`

	// Description is a short high level summary of the status.
	Description string
}

type GitPullRequestArguments struct {
	Title             string
	Body              string
	Head              string
	Base              string
	GitRepositoryInfo *GitRepositoryInfo
}

type GitWebHookArguments struct {
	ID     int64
	Owner  string
	Repo   *GitRepositoryInfo
	URL    string
	Secret string
}

type GitFileContent struct {
	Type        string
	Encoding    string
	Size        int
	Name        string
	Path        string
	Content     string
	Sha         string
	Url         string
	GitUrl      string
	HtmlUrl     string
	DownloadUrl string
}

// IsClosed returns true if the PullRequest has been closed
func (pr *GitPullRequest) IsClosed() bool {
	return pr.ClosedAt != nil
}

// NumberString returns the string representation of the Pull Request number or blank if its missing
func (pr *GitPullRequest) NumberString() string {
	n := pr.Number
	if n == nil {
		return ""
	}
	return "#" + strconv.Itoa(*n)
}

func CreateProvider(server *auth.AuthServer, user *auth.UserAuth, git Gitter) (GitProvider, error) {
	if (server.Kind == KindBitBucketCloud) || (server.Kind == "" && strings.HasPrefix(server.URL, "https://bitbucket.org")) {
		return NewBitbucketCloudProvider(server, user, git)
	} else if server.Kind == KindBitBucketServer {
		return NewBitbucketServerProvider(server, user, git)
	} else if server.Kind == KindGitea {
		return NewGiteaProvider(server, user, git)
	} else if server.Kind == KindGitlab {
		return NewGitlabProvider(server, user, git)
	} else {
		return NewGitHubProvider(server, user, git)
	}
}

// GetHost returns the Git Provider hostname, e.g github.com
func GetHost(gitProvider GitProvider) (string, error) {
	if gitProvider == nil {
		return "", fmt.Errorf("no Git provider")
	}

	if gitProvider.ServerURL() == "" {
		return "", fmt.Errorf("no Git provider server URL found")
	}
	url, err := url.Parse(gitProvider.ServerURL())
	if err != nil {
		return "", fmt.Errorf("error parsing ")
	}
	return url.Host, nil
}

func ProviderAccessTokenURL(kind string, url string, username string) string {
	switch kind {
	case KindBitBucketCloud:
		// TODO pass in the username
		return BitBucketCloudAccessTokenURL(url, username)
	case KindBitBucketServer:
		return BitBucketServerAccessTokenURL(url)
	case KindGitea:
		return GiteaAccessTokenURL(url)
	case KindGitlab:
		return GitlabAccessTokenURL(url)
	default:
		return GitHubAccessTokenURL(url)
	}
}

// PickOrganisation picks an organisations login if there is one available
func PickOrganisation(orgLister OrganisationLister, userName string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (string, error) {
	prompt := &survey.Select{
		Message: "Which organisation do you want to use?",
		Options: GetOrganizations(orgLister, userName),
		Default: userName,
	}

	orgName := ""
	surveyOpts := survey.WithStdio(in, out, errOut)
	err := survey.AskOne(prompt, &orgName, nil, surveyOpts)
	if err != nil {
		return "", err
	}
	if orgName == userName {
		return "", nil
	}
	return orgName, nil
}

// GetOrganizations gets the organisation
func GetOrganizations(orgLister OrganisationLister, userName string) []string {
	// Always include the username as a pseudo organization
	orgNames := []string{userName}

	orgs, _ := orgLister.ListOrganisations()
	for _, o := range orgs {
		if name := o.Login; name != "" {
			orgNames = append(orgNames, name)
		}
	}
	sort.Strings(orgNames)
	return orgNames
}

func PickRepositories(provider GitProvider, owner string, message string, selectAll bool, filter string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) ([]*GitRepository, error) {
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
	surveyOpts := survey.WithStdio(in, out, errOut)
	err = survey.AskOne(prompt, &repoNames, nil, surveyOpts)

	for _, n := range repoNames {
		repo := repoMap[n]
		if repo != nil {
			answer = append(answer, repo)
		}
	}
	return answer, err
}

// IsGitRepoStatusSuccess returns true if all the statuses are successful
func IsGitRepoStatusSuccess(statuses ...*GitRepoStatus) bool {
	for _, status := range statuses {
		if !status.IsSuccess() {
			return false
		}
	}
	return true
}

// IsGitRepoStatusFailed returns true if any of the statuses have failed
func IsGitRepoStatusFailed(statuses ...*GitRepoStatus) bool {
	for _, status := range statuses {
		if status.IsFailed() {
			return true
		}
	}
	return false
}

func (s *GitRepoStatus) IsSuccess() bool {
	return s.State == "success"
}

func (s *GitRepoStatus) IsFailed() bool {
	return s.State == "error" || s.State == "failure"
}

func (i *GitRepositoryInfo) PickOrCreateProvider(authConfigSvc auth.AuthConfigService, message string, batchMode bool, gitKind string, git Gitter, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (GitProvider, error) {
	config := authConfigSvc.Config()
	hostUrl := i.HostURLWithoutUser()
	server := config.GetOrCreateServer(hostUrl)
	if server.Kind == "" {
		server.Kind = gitKind
	}
	userAuth, err := config.PickServerUserAuth(server, message, batchMode, "", in, out, errOut)
	if err != nil {
		return nil, err
	}
	return i.CreateProviderForUser(server, userAuth, gitKind, git)
}

func (i *GitRepositoryInfo) CreateProviderForUser(server *auth.AuthServer, user *auth.UserAuth, gitKind string, git Gitter) (GitProvider, error) {
	if i.Host == GitHubHost {
		return NewGitHubProvider(server, user, git)
	}
	if gitKind != "" && server.Kind != gitKind {
		server.Kind = gitKind
	}
	return CreateProvider(server, user, git)
}

func (i *GitRepositoryInfo) CreateProvider(authConfigSvc auth.AuthConfigService, gitKind string, git Gitter, batchMode bool, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (GitProvider, error) {
	hostUrl := i.HostURLWithoutUser()
	return CreateProviderForURL(authConfigSvc, gitKind, hostUrl, git, batchMode, in, out, errOut)
}

// CreateProviderForURL creates the Git provider for the given git kind and host URL
func CreateProviderForURL(authConfigSvc auth.AuthConfigService, gitKind string, hostUrl string, git Gitter, batchMode bool, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (GitProvider, error) {
	config := authConfigSvc.Config()
	server := config.GetOrCreateServer(hostUrl)
	url := server.URL
	if gitKind != "" {
		server.Kind = gitKind
	}
	userAuths := authConfigSvc.Config().FindUserAuths(url)
	if len(userAuths) == 0 {
		kind := server.Kind
		if kind != "" {
			userAuth := auth.CreateAuthUserFromEnvironment(strings.ToUpper(kind))
			if !userAuth.IsInvalid() {
				return CreateProvider(server, &userAuth, git)
			}
		}
		userAuth := auth.CreateAuthUserFromEnvironment("GIT")
		if !userAuth.IsInvalid() {
			return CreateProvider(server, &userAuth, git)
		}
	}
	if len(userAuths) > 0 {
		// TODO use default user???
		auth := userAuths[0]
		return CreateProvider(server, auth, git)
	}
	auth, err := createUserForServer(batchMode, authConfigSvc, server, git, in, out, errOut)
	if err != nil {
		return nil, err
	}
	return CreateProvider(server, auth, git)
}

func createUserForServer(batchMode bool, authConfigSvc auth.AuthConfigService, server *auth.AuthServer,
	git Gitter, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) (*auth.UserAuth, error) {
	userAuth := &auth.UserAuth{}

	f := func(username string) error {
		git.PrintCreateRepositoryGenerateAccessToken(server, username, out)
		return nil
	}

	// TODO could we guess this based on the users ~/.git for github?
	defaultUserName := ""
	err := authConfigSvc.Config().EditUserAuth(server.Label(), userAuth, defaultUserName, false, batchMode, f, in, out, errOut)
	if err != nil {
		return userAuth, err
	}

	// TODO lets verify the auth works

	err = authConfigSvc.SaveUserAuth(server.URL, userAuth)
	if err != nil {
		return userAuth, fmt.Errorf("failed to store git auth configuration %s", err)
	}
	if userAuth.IsInvalid() {
		return userAuth, fmt.Errorf("you did not properly define the user authentication")
	}
	return userAuth, nil
}
