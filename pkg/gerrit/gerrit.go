package gerrit

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/andygrunwald/go-gerrit"
	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/wbrefvem/go-gits/pkg/git"
)

type GerritProvider struct {
	Client   *gerrit.Client
	Username string
	Context  context.Context

	Server auth.AuthServer
	User   auth.UserAuth
	Git    git.Gitter
}

func NewGerritProvider(server *auth.AuthServer, user *auth.UserAuth, git git.Gitter) (git.GitProvider, error) {
	ctx := context.Background()

	provider := GerritProvider{
		Server:   *server,
		User:     *user,
		Context:  ctx,
		Username: user.Username,
		Git:      git,
	}

	client, err := gerrit.NewClient(server.URL, nil)
	if err != nil {
		return nil, err
	}
	client.Authentication.SetBasicAuth(user.Username, user.ApiToken)
	provider.Client = client

	return &provider, nil
}

// We have to do this because url.Escape is not idempotent, so we unescape the URL
// to ensure it's not encoded, then we re-encode it.
func buildEncodedProjectName(org, name string) string {
	var fullName string

	if org != "" {
		fullName = fmt.Sprintf("%s/%s", org, name)
	} else {
		fullName = fmt.Sprintf("%s", name)
	}

	fullNamePathUnescaped, err := url.PathUnescape(fullName)
	if err != nil {
		return ""
	}
	fullNamePathEscaped := url.PathEscape(fullNamePathUnescaped)

	return fullNamePathEscaped
}

func (p *GerritProvider) projectInfoToGitRepository(project *gerrit.ProjectInfo) *git.GitRepository {
	return &git.GitRepository{
		Name:     project.Name,
		CloneURL: fmt.Sprintf("%s/%s", p.Server.URL, project.Name),
		SSHURL:   fmt.Sprintf("%s:%s", p.Server.URL, project.Name),
	}
}

func (p *GerritProvider) ListRepositories(org string) ([]*git.GitRepository, error) {
	options := &gerrit.ProjectOptions{
		Description: true,
		Prefix:      url.PathEscape(org),
	}

	gerritProjects, _, err := p.Client.Projects.ListProjects(options)
	if err != nil {
		return nil, err
	}

	repos := []*git.GitRepository{}

	for name, project := range *gerritProjects {
		project.Name = name
		repo := p.projectInfoToGitRepository(&project)

		repos = append(repos, repo)
	}

	return repos, nil
}

func (p *GerritProvider) CreateRepository(org string, name string, private bool) (*git.GitRepository, error) {
	input := &gerrit.ProjectInput{
		SubmitType:      "INHERIT",
		Description:     "Created automatically by Jenkins X.",
		PermissionsOnly: private,
	}

	fullNamePathEscaped := buildEncodedProjectName(org, name)
	project, _, err := p.Client.Projects.CreateProject(fullNamePathEscaped, input)
	if err != nil {
		return nil, err
	}

	repo := p.projectInfoToGitRepository(project)
	return repo, nil
}

func (p *GerritProvider) GetRepository(org string, name string) (*git.GitRepository, error) {
	fullName := buildEncodedProjectName(org, name)

	project, _, err := p.Client.Projects.GetProject(fullName)
	if err != nil {
		return nil, err
	}
	return p.projectInfoToGitRepository(project), nil
}

func (p *GerritProvider) DeleteRepository(org string, name string) error {
	return nil
}

func (p *GerritProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*git.GitRepository, error) {
	return nil, nil
}

func (p *GerritProvider) RenameRepository(org string, name string, newName string) (*git.GitRepository, error) {
	return nil, nil
}

func (p *GerritProvider) ValidateRepositoryName(org string, name string) error {
	return nil
}

func (p *GerritProvider) CreatePullRequest(data *git.GitPullRequestArguments) (*git.GitPullRequest, error) {
	return nil, nil
}

func (p *GerritProvider) UpdatePullRequestStatus(pr *git.GitPullRequest) error {
	return nil
}

func (p *GerritProvider) GetPullRequest(owner string, repo *git.GitRepository, number int) (*git.GitPullRequest, error) {
	return nil, nil
}

func (p *GerritProvider) GetPullRequestCommits(owner string, repo *git.GitRepository, number int) ([]*git.GitCommit, error) {
	return nil, nil
}

func (p *GerritProvider) PullRequestLastCommitStatus(pr *git.GitPullRequest) (string, error) {
	return "", nil
}

func (p *GerritProvider) ListCommitStatus(org string, repo string, sha string) ([]*git.GitRepoStatus, error) {
	return nil, nil
}

// UpdateCommitStatus updates the status of a specified commit in a specified repo.
func (p *GerritProvider) UpdateCommitStatus(org, repo, sha string, status *git.GitRepoStatus) (*git.GitRepoStatus, error) {
	return nil, nil
}

func (p *GerritProvider) MergePullRequest(pr *git.GitPullRequest, message string) error {
	return nil
}

func (p *GerritProvider) CreateWebHook(data *git.GitWebHookArguments) error {
	return nil
}

// UpdateWebHook update a webhook with the data specified.
func (p *GerritProvider) UpdateWebHook(data *git.GitWebHookArguments) error {
	return nil
}

// ListWebHooks lists all webhooks for the specified repo.
func (p *GerritProvider) ListWebHooks(org, repo string) ([]*git.GitWebHookArguments, error) {
	return nil, nil
}

// ListOrganisations lists all organizations the configured user has access to.
func (p *GerritProvider) ListOrganisations() ([]git.GitOrganisation, error) {
	return nil, nil
}

func (p *GerritProvider) IsGitHub() bool {
	return false
}

func (p *GerritProvider) IsGitea() bool {
	return false
}

func (p *GerritProvider) IsBitbucketCloud() bool {
	return false
}

func (p *GerritProvider) IsBitbucketServer() bool {
	return false
}

func (p *GerritProvider) IsGerrit() bool {
	return true
}

func (p *GerritProvider) Kind() string {
	return "gerrit"
}

func (p *GerritProvider) GetIssue(org string, name string, number int) (*git.GitIssue, error) {
	log.Warn("Gerrit does not support issue tracking")
	return nil, nil
}

func (p *GerritProvider) IssueURL(org string, name string, number int, isPull bool) string {
	log.Warn("Gerrit does not support issue tracking")
	return ""
}

func (p *GerritProvider) SearchIssues(org string, name string, query string) ([]*git.GitIssue, error) {
	log.Warn("Gerrit does not support issue tracking")
	return nil, nil
}

func (p *GerritProvider) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*git.GitIssue, error) {
	log.Warn("Gerrit does not support issue tracking")
	return nil, nil
}

func (p *GerritProvider) CreateIssue(owner string, repo string, issue *git.GitIssue) (*git.GitIssue, error) {
	log.Warn("Gerrit does not support issue tracking")
	return nil, nil
}

func (p *GerritProvider) HasIssues() bool {
	log.Warn("Gerrit does not support issue tracking")
	return false
}

func (p *GerritProvider) AddPRComment(pr *git.GitPullRequest, comment string) error {
	return nil
}

func (p *GerritProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	log.Warn("Gerrit does not support issue tracking")
	return nil
}

func (p *GerritProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *git.GitRelease) error {
	return nil
}

func (p *GerritProvider) ListReleases(org string, name string) ([]*git.GitRelease, error) {
	return nil, nil
}

func (p *GerritProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return ""
}

func (p *GerritProvider) Label() string {
	return ""
}

func (p *GerritProvider) ServerURL() string {
	return ""
}

func (p *GerritProvider) BranchArchiveURL(org string, name string, branch string) string {
	return ""
}

func (p *GerritProvider) CurrentUsername() string {
	return ""
}

func (p *GerritProvider) UserAuth() auth.UserAuth {
	return auth.UserAuth{}
}

func (p *GerritProvider) UserInfo(username string) *git.GitUser {
	return nil
}

func (p *GerritProvider) AddCollaborator(user string, organisation string, repo string) error {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for gerrit. Please add user: %v as a collaborator to this project.\n", user)
	return nil
}

func (p *GerritProvider) ListInvitations() ([]*github.RepositoryInvitation, *github.Response, error) {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for gerrit.\n")
	return []*github.RepositoryInvitation{}, &github.Response{}, nil
}

func (p *GerritProvider) AcceptInvitation(ID int64) (*github.Response, error) {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for gerrit.\n")
	return &github.Response{}, nil
}

func (p *GerritProvider) GetContent(org string, name string, path string, ref string) (*git.GitFileContent, error) {
	return nil, fmt.Errorf("Getting content not supported on gerrit")
}

func (p *GerritProvider) AccessTokenURL() string {
	return ""
}
