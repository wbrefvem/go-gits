package gerrit

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/andygrunwald/go-gerrit"
	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/wbrefvem/go-gits/pkg/git"
)

type GerritProvider struct {
	Client   *gerrit.Client
	Username string
	Context  context.Context

	Git git.Gitter
}

func NewProvider(git git.Gitter) (git.Provider, error) {
	ctx := context.Background()

	provider := GerritProvider{
		Context: ctx,
		Git:     git,
	}

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

func (p *GerritProvider) projectInfoToGitRepository(project *gerrit.ProjectInfo) *git.Repository {
	return &git.Repository{
		Name: project.Name,
	}
}

func (p *GerritProvider) ListRepositories(org string) ([]*git.Repository, error) {
	options := &gerrit.ProjectOptions{
		Description: true,
		Prefix:      url.PathEscape(org),
	}

	gerritProjects, _, err := p.Client.Projects.ListProjects(options)
	if err != nil {
		return nil, err
	}

	repos := []*git.Repository{}

	for name, project := range *gerritProjects {
		project.Name = name
		repo := p.projectInfoToGitRepository(&project)

		repos = append(repos, repo)
	}

	return repos, nil
}

func (p *GerritProvider) CreateRepository(org string, name string, private bool) (*git.Repository, error) {
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

func (p *GerritProvider) GetRepository(org string, name string) (*git.Repository, error) {
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

func (p *GerritProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*git.Repository, error) {
	return nil, nil
}

func (p *GerritProvider) RenameRepository(org string, name string, newName string) (*git.Repository, error) {
	return nil, nil
}

func (p *GerritProvider) ValidateRepositoryName(org string, name string) error {
	return nil
}

func (p *GerritProvider) CreatePullRequest(data *git.PullRequestArguments) (*git.PullRequest, error) {
	return nil, nil
}

func (p *GerritProvider) UpdatePullRequestStatus(pr *git.PullRequest) error {
	return nil
}

func (p *GerritProvider) GetPullRequest(owner string, repo *git.Repository, number int) (*git.PullRequest, error) {
	return nil, nil
}

func (p *GerritProvider) GetPullRequestCommits(owner string, repo *git.Repository, number int) ([]*git.Commit, error) {
	return nil, nil
}

func (p *GerritProvider) PullRequestLastCommitStatus(pr *git.PullRequest) (string, error) {
	return "", nil
}

func (p *GerritProvider) ListCommitStatus(org string, repo string, sha string) ([]*git.RepoStatus, error) {
	return nil, nil
}

// UpdateCommitStatus updates the status of a specified commit in a specified repo.
func (p *GerritProvider) UpdateCommitStatus(org, repo, sha string, status *git.RepoStatus) (*git.RepoStatus, error) {
	return nil, nil
}

func (p *GerritProvider) MergePullRequest(pr *git.PullRequest, message string) error {
	return nil
}

func (p *GerritProvider) CreateWebHook(data *git.WebhookArguments) error {
	return nil
}

// UpdateWebHook update a webhook with the data specified.
func (p *GerritProvider) UpdateWebHook(data *git.WebhookArguments) error {
	return nil
}

// ListWebHooks lists all webhooks for the specified repo.
func (p *GerritProvider) ListWebHooks(org, repo string) ([]*git.WebhookArguments, error) {
	return nil, nil
}

// ListOrganisations lists all organizations the configured user has access to.
func (p *GerritProvider) ListOrganisations() ([]git.Organisation, error) {
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

func (p *GerritProvider) GetIssue(org string, name string, number int) (*git.Issue, error) {
	log.Warn("Gerrit does not support issue tracking")
	return nil, nil
}

func (p *GerritProvider) IssueURL(org string, name string, number int, isPull bool) string {
	log.Warn("Gerrit does not support issue tracking")
	return ""
}

func (p *GerritProvider) SearchIssues(org string, name string, query string) ([]*git.Issue, error) {
	log.Warn("Gerrit does not support issue tracking")
	return nil, nil
}

func (p *GerritProvider) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*git.Issue, error) {
	log.Warn("Gerrit does not support issue tracking")
	return nil, nil
}

func (p *GerritProvider) CreateIssue(owner string, repo string, issue *git.Issue) (*git.Issue, error) {
	log.Warn("Gerrit does not support issue tracking")
	return nil, nil
}

func (p *GerritProvider) HasIssues() bool {
	log.Warn("Gerrit does not support issue tracking")
	return false
}

func (p *GerritProvider) AddPRComment(pr *git.PullRequest, comment string) error {
	return nil
}

func (p *GerritProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	log.Warn("Gerrit does not support issue tracking")
	return nil
}

func (p *GerritProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *git.Release) error {
	return nil
}

func (p *GerritProvider) ListReleases(org string, name string) ([]*git.Release, error) {
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

func (p *GerritProvider) UserInfo(username string) *git.User {
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

func (p *GerritProvider) GetContent(org string, name string, path string, ref string) (*git.FileContent, error) {
	return nil, fmt.Errorf("Getting content not supported on gerrit")
}

func (p *GerritProvider) AccessTokenURL() string {
	return ""
}
