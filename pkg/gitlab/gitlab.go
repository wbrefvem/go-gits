package gitlab

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/xanzy/go-gitlab"

	"github.com/wbrefvem/go-gits/pkg/git"
)

type GitlabProvider struct {
	Username string
	Client   *gitlab.Client
	Context  context.Context

	URL  string
	Git  git.Gitter
	Name string
}

func NewGitlabProvider(username, serverURL, token, providerName string, git git.Gitter) (git.Provider, error) {
	u := serverURL
	c := gitlab.NewClient(nil, username)
	if !IsGitLabServerURL(u) {
		if err := c.SetBaseURL(u); err != nil {
			return nil, err
		}
	}
	return WithGitlabClient(serverURL, username, c, git)
}

func IsGitLabServerURL(u string) bool {
	u = strings.TrimSuffix(u, "/")
	return u == "" || u == "https://gitlab.com" || u == "http://gitlab.com"
}

// Used by unit tests to inject a mocked client
func WithGitlabClient(serverURL, username string, client *gitlab.Client, git git.Gitter) (git.Provider, error) {
	provider := &GitlabProvider{
		Username: username,
		Client:   client,
		Git:      git,
		URL:      serverURL,
	}
	return provider, nil
}

func (g *GitlabProvider) ListRepositories(org string) ([]*git.Repository, error) {
	result, _, err := getRepositories(g.Client, g.Username, org)
	if err != nil {
		return nil, err
	}

	var repos []*git.Repository
	for _, p := range result {
		repos = append(repos, fromGitlabProject(p))
	}
	return repos, nil
}

func (g *GitlabProvider) ListReleases(org string, name string) ([]*git.Release, error) {
	answer := []*git.Release{}
	// TODO
	return answer, nil
}

func getRepositories(g *gitlab.Client, username string, org string) ([]*gitlab.Project, *gitlab.Response, error) {
	if org != "" {
		projects, resp, err := g.Groups.ListGroupProjects(org, nil)
		if err != nil {
			return g.Projects.ListUserProjects(org, &gitlab.ListProjectsOptions{Owned: gitlab.Bool(true)})
		}
		return projects, resp, err

	}
	return g.Projects.ListUserProjects(username, &gitlab.ListProjectsOptions{Owned: gitlab.Bool(true)})
}

func fromGitlabProject(p *gitlab.Project) *git.Repository {
	return &git.Repository{
		Name:     p.Name,
		HTMLURL:  p.WebURL,
		SSHURL:   p.SSHURLToRepo,
		CloneURL: p.HTTPURLToRepo,
		Fork:     p.ForkedFromProject != nil,
	}
}

func (g *GitlabProvider) CreateRepository(org string, name string, private bool) (*git.Repository, error) {
	visibility := gitlab.PublicVisibility
	if private {
		visibility = gitlab.PrivateVisibility
	}

	p := &gitlab.CreateProjectOptions{
		Name:       &name,
		Visibility: &visibility,
	}

	project, _, err := g.Client.Projects.CreateProject(p)
	if err != nil {
		return nil, err
	}
	return fromGitlabProject(project), nil
}

func owner(org, username string) string {
	if org == "" {
		return username
	}
	return org
}

func (g *GitlabProvider) GetRepository(org, name string) (*git.Repository, error) {
	pid, err := g.projectId(org, g.Username, name)
	if err != nil {
		return nil, err
	}
	project, response, err := g.Client.Projects.GetProject(pid)
	if err != nil {
		return nil, fmt.Errorf("request: %s failed due to: %s", response.Request.URL, err)
	}
	return fromGitlabProject(project), nil
}

func (g *GitlabProvider) ListOrganisations() ([]git.Organisation, error) {
	groups, _, err := g.Client.Groups.ListGroups(nil)
	if err != nil {
		return nil, err
	}

	var organizations []git.Organisation
	for _, v := range groups {
		organizations = append(organizations, git.Organisation{v.Path})
	}
	return organizations, nil
}

func (g *GitlabProvider) projectId(org, username, name string) (string, error) {
	repos, _, err := getRepositories(g.Client, username, org)
	if err != nil {
		return "", err
	}

	for _, repo := range repos {
		if repo.Name == name {
			return strconv.Itoa(repo.ID), nil
		}
	}
	return "", fmt.Errorf("no repository found with name %s", name)
}

func (g *GitlabProvider) DeleteRepository(org, name string) error {
	pid, err := g.projectId(org, g.Username, name)
	if err != nil {
		return err
	}

	_, err = g.Client.Projects.DeleteProject(pid)
	if err != nil {
		return fmt.Errorf("failed to delete repository %s due to: %s", pid, err)
	}
	return err
}

func (g *GitlabProvider) ForkRepository(originalOrg, name, destinationOrg string) (*git.Repository, error) {
	pid, err := g.projectId(originalOrg, g.Username, name)
	if err != nil {
		return nil, err
	}
	project, _, err := g.Client.Projects.ForkProject(pid)
	if err != nil {
		return nil, err
	}

	return fromGitlabProject(project), nil
}

func (g *GitlabProvider) RenameRepository(org, name, newName string) (*git.Repository, error) {
	pid, err := g.projectId(org, g.Username, name)
	if err != nil {
		return nil, err
	}
	options := &gitlab.EditProjectOptions{
		Name: &newName,
	}

	project, _, err := g.Client.Projects.EditProject(pid, options)
	if err != nil {
		return nil, err
	}
	return fromGitlabProject(project), nil
}

func (g *GitlabProvider) ValidateRepositoryName(org, name string) error {
	pid, err := g.projectId(org, g.Username, name)
	if err == nil {
		return fmt.Errorf("repository %s already exists", pid)
	}
	return nil
}

func (g *GitlabProvider) CreatePullRequest(data *git.PullRequestArguments) (*git.PullRequest, error) {
	owner := data.Repository.Organisation
	repo := data.Repository.Name
	title := data.Title
	body := data.Body
	head := data.Head
	base := data.Base

	o := &gitlab.CreateMergeRequestOptions{
		Title:        &title,
		Description:  &body,
		SourceBranch: &head,
		TargetBranch: &base,
	}

	pid, err := g.projectId(owner, g.Username, repo)
	if err != nil {
		return nil, err
	}
	mr, _, err := g.Client.MergeRequests.CreateMergeRequest(pid, o)
	if err != nil {
		return nil, err
	}

	return fromMergeRequest(mr, owner, repo), nil
}

func fromMergeRequest(mr *gitlab.MergeRequest, owner, repo string) *git.PullRequest {
	merged := false
	if mr.MergedAt != nil {
		merged = true
	}
	return &git.PullRequest{
		Author: &git.User{
			Login: mr.Author.Username,
		},
		URL:            mr.WebURL,
		Owner:          owner,
		Repo:           repo,
		Number:         &mr.IID,
		State:          &mr.State,
		Title:          mr.Title,
		Body:           mr.Description,
		MergeCommitSHA: &mr.MergeCommitSHA,
		Merged:         &merged,
		LastCommitSha:  mr.SHA,
		MergedAt:       mr.MergedAt,
		ClosedAt:       mr.ClosedAt,
	}
}

func (g *GitlabProvider) UpdatePullRequestStatus(pr *git.PullRequest) error {
	owner := pr.Owner
	repo := pr.Repo

	pid, err := g.projectId(owner, g.Username, repo)
	if err != nil {
		return err
	}
	mr, _, err := g.Client.MergeRequests.GetMergeRequest(pid, *pr.Number)
	if err != nil {
		return err
	}

	*pr = *fromMergeRequest(mr, owner, repo)
	return nil
}

func (p *GitlabProvider) GetPullRequest(owner string, repo *git.Repository, number int) (*git.PullRequest, error) {
	pr := &git.PullRequest{
		Owner:  owner,
		Repo:   repo.Name,
		Number: &number,
	}
	err := p.UpdatePullRequestStatus(pr)

	existing := p.UserInfo(pr.Author.Login)
	if existing != nil && existing.Email != "" {
		pr.Author = existing
	}

	return pr, err
}

func (p *GitlabProvider) GetPullRequestCommits(owner string, repository *git.Repository, number int) ([]*git.Commit, error) {
	repo := repository.Name
	pid, err := p.projectId(owner, p.Username, repo)
	if err != nil {
		return nil, err
	}
	commits, _, err := p.Client.MergeRequests.GetMergeRequestCommits(pid, number, nil)

	if err != nil {
		return nil, err
	}

	answer := []*git.Commit{}

	for _, commit := range commits {
		if commit == nil {
			continue
		}
		summary := &git.Commit{
			Message: commit.Message,
			SHA:     commit.ID,
			Author: &git.User{
				Email: commit.AuthorEmail,
			},
		}
		answer = append(answer, summary)
	}

	return answer, nil
}

func (g *GitlabProvider) PullRequestLastCommitStatus(pr *git.PullRequest) (string, error) {
	owner := pr.Owner
	repo := pr.Repo

	ref := pr.LastCommitSha
	if ref == "" {
		return "", fmt.Errorf("missing String for LastCommitSha %#v", pr)
	}

	pid, err := g.projectId(owner, g.Username, repo)
	if err != nil {
		return "", err
	}
	c, _, err := g.Client.Commits.GetCommitStatuses(pid, ref, nil)
	if err != nil {
		return "", err
	}

	for _, result := range c {
		if result.Status != "" {
			return result.Status, nil
		}
	}
	return "", fmt.Errorf("could not find a status for repository %s with ref %s", pid, ref)
}

func (g *GitlabProvider) ListCommitStatus(org string, repo string, sha string) ([]*git.RepoStatus, error) {
	pid, err := g.projectId(org, g.Username, repo)
	if err != nil {
		return nil, err
	}
	c, _, err := g.Client.Commits.GetCommitStatuses(pid, sha, nil)
	if err != nil {
		return nil, err
	}

	var statuses []*git.RepoStatus

	for _, result := range c {
		statuses = append(statuses, fromCommitStatus(result))
	}

	return statuses, nil
}

func (b *GitlabProvider) UpdateCommitStatus(org string, repo string, sha string, status *git.RepoStatus) (*git.RepoStatus, error) {
	return &git.RepoStatus{}, errors.New("TODO")
}

func fromCommitStatus(status *gitlab.CommitStatus) *git.RepoStatus {
	return &git.RepoStatus{
		ID:          string(status.ID),
		URL:         status.TargetURL,
		State:       status.Status,
		Description: status.Description,
	}
}

func (g *GitlabProvider) MergePullRequest(pr *git.PullRequest, message string) error {
	pid, err := g.projectId(pr.Owner, g.Username, pr.Repo)
	if err != nil {
		return err
	}

	opt := &gitlab.AcceptMergeRequestOptions{MergeCommitMessage: &message}

	_, _, err = g.Client.MergeRequests.AcceptMergeRequest(pid, *pr.Number, opt)
	return err
}

func (g *GitlabProvider) CreateWebHook(data *git.WebhookArguments) error {
	pid, err := g.projectId(data.Owner, g.Username, data.Repo.Name)
	if err != nil {
		return nil
	}

	owner := owner(g.Username, data.Owner)
	webhookURL := util.UrlJoin(data.URL, owner, data.Repo.Name)
	opt := &gitlab.AddProjectHookOptions{
		URL:   &webhookURL,
		Token: &data.Secret,
	}

	_, _, err = g.Client.Projects.AddProjectHook(pid, opt)
	return err
}

func (p *GitlabProvider) ListWebHooks(owner string, repo string) ([]*git.WebhookArguments, error) {
	webHooks := []*git.WebhookArguments{}
	return webHooks, fmt.Errorf("not implemented!")
}

func (p *GitlabProvider) UpdateWebHook(data *git.WebhookArguments) error {
	return fmt.Errorf("not implemented!")
}

func (g *GitlabProvider) SearchIssues(org, repo, query string) ([]*git.Issue, error) {
	opt := &gitlab.ListProjectIssuesOptions{Search: &query}
	return g.searchIssuesWithOptions(org, repo, opt)
}

func (g *GitlabProvider) SearchIssuesClosedSince(org string, repo string, t time.Time) ([]*git.Issue, error) {
	closed := "closed"
	opt := &gitlab.ListProjectIssuesOptions{State: &closed}
	issues, err := g.searchIssuesWithOptions(org, repo, opt)
	if err != nil {
		return issues, err
	}
	return git.FilterIssuesClosedSince(issues, t), nil
}

func (g *GitlabProvider) searchIssuesWithOptions(org string, repo string, opt *gitlab.ListProjectIssuesOptions) ([]*git.Issue, error) {
	pid, err := g.projectId(org, g.Username, repo)
	if err != nil {
		return nil, err
	}
	issues, _, err := g.Client.Issues.ListProjectIssues(pid, opt)
	if err != nil {
		return nil, err
	}
	return fromGitlabIssues(issues, owner(org, g.Username), repo), nil
}

func (g *GitlabProvider) GetIssue(org, repo string, number int) (*git.Issue, error) {
	owner := owner(org, g.Username)
	pid, err := g.projectId(org, g.Username, repo)
	if err != nil {
		return nil, err
	}

	issue, _, err := g.Client.Issues.GetIssue(pid, number)
	if err != nil {
		return nil, err
	}
	return fromGitlabIssue(issue, owner, repo), nil
}

func (g *GitlabProvider) CreateIssue(owner string, repo string, issue *git.Issue) (*git.Issue, error) {
	labels := []string{}
	for _, label := range issue.Labels {
		name := label.Name
		if name != "" {
			labels = append(labels, name)
		}
	}

	opt := &gitlab.CreateIssueOptions{
		Title:       &issue.Title,
		Description: &issue.Body,
		Labels:      labels,
	}

	pid, err := g.projectId(owner, g.Username, repo)
	if err != nil {
		return nil, err
	}
	gitlabIssue, _, err := g.Client.Issues.CreateIssue(pid, opt)
	if err != nil {
		return nil, err
	}

	return fromGitlabIssue(gitlabIssue, owner, repo), nil
}

func fromGitlabIssues(issues []*gitlab.Issue, owner, repo string) []*git.Issue {
	var result []*git.Issue

	for _, v := range issues {
		result = append(result, fromGitlabIssue(v, owner, repo))
	}
	return result
}

func fromGitlabIssue(issue *gitlab.Issue, owner, repo string) *git.Issue {
	var labels []git.Label
	for _, v := range issue.Labels {
		labels = append(labels, git.Label{Name: v})
	}

	return &git.Issue{
		Number:    &issue.IID,
		URL:       issue.WebURL,
		Owner:     owner,
		Repo:      repo,
		Title:     issue.Title,
		Body:      issue.Description,
		Labels:    labels,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
		ClosedAt:  issue.ClosedAt,
	}
}

func (g *GitlabProvider) AddPRComment(pr *git.PullRequest, comment string) error {
	owner := pr.Owner
	repo := pr.Repo

	opt := &gitlab.CreateMergeRequestNoteOptions{Body: &comment}

	pid, err := g.projectId(owner, g.Username, repo)
	if err != nil {
		return nil
	}
	_, _, err = g.Client.Notes.CreateMergeRequestNote(pid, *pr.Number, opt)
	return err
}

func (g *GitlabProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	opt := &gitlab.CreateIssueNoteOptions{Body: &comment}

	pid, err := g.projectId(owner, g.Username, repo)
	if err != nil {
		return err
	}
	_, _, err = g.Client.Notes.CreateIssueNote(pid, number, opt)
	return err
}

func (g *GitlabProvider) HasIssues() bool {
	return true
}

func (g *GitlabProvider) IsGitHub() bool {
	return false
}

func (g *GitlabProvider) IsGitea() bool {
	return false
}

func (g *GitlabProvider) IsBitbucketCloud() bool {
	return false
}

func (g *GitlabProvider) IsBitbucketServer() bool {
	return false
}

func (g *GitlabProvider) IsGerrit() bool {
	return false
}

func (g *GitlabProvider) Kind() string {
	return "gitlab"
}

func (g *GitlabProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/project"
}

func (g *GitlabProvider) Label() string {
	return g.Name
}

func (p *GitlabProvider) ServerURL() string {
	return p.URL
}

func (p *GitlabProvider) BranchArchiveURL(org string, name string, branch string) string {
	return util.UrlJoin(p.ServerURL(), org, name, "-/archive", branch, name+"-"+branch+".zip")
}

func (p *GitlabProvider) CurrentUsername() string {
	return p.Username
}

func (p *GitlabProvider) UserInfo(username string) *git.User {
	users, _, err := p.Client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &username})

	if err != nil || len(users) == 0 {
		return nil
	}

	user := users[0]

	return &git.User{
		Login:     username,
		URL:       user.WebsiteURL,
		AvatarURL: user.AvatarURL,
		Name:      user.Name,
		Email:     user.Email,
	}
}

func (g *GitlabProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *git.Release) error {
	return nil
}

func (p *GitlabProvider) IssueURL(org string, name string, number int, isPull bool) string {
	return ""
}

func (p *GitlabProvider) AddCollaborator(user string, organisation string, repo string) error {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for gitlab. Please add user: %v as a collaborator to this project.\n", user)
	return nil
}

func (p *GitlabProvider) GetContent(org string, name string, path string, ref string) (*git.FileContent, error) {
	return nil, fmt.Errorf("Getting content not supported on gitlab")
}

// GitlabAccessTokenURL returns the URL to click on to generate a personal access token for the Git provider
func (p *GitlabProvider) AccessTokenURL() string {
	return util.UrlJoin(p.ServerURL(), "/profile/personal_access_tokens")
}
