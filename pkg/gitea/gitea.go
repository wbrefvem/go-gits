package gitea

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/wbrefvem/go-gits/pkg/git"
)

type GiteaProvider struct {
	Username string
	Client   *gitea.Client
	URL      string
	Git      git.Gitter
	Name     string
}

func NewGiteaProvider(username, serverURL, token, providerName string, git git.Gitter) (git.Provider, error) {
	client := gitea.NewClient(serverURL, token)

	provider := GiteaProvider{
		Client:   client,
		Username: username,
		Git:      git,
		Name:     providerName,
	}

	return &provider, nil
}

func (p *GiteaProvider) ListOrganisations() ([]git.Organisation, error) {
	answer := []git.Organisation{}
	orgs, err := p.Client.ListMyOrgs()
	if err != nil {
		return answer, err
	}

	for _, org := range orgs {
		name := org.UserName
		if name != "" {
			o := git.Organisation{
				Login: name,
			}
			answer = append(answer, o)
		}
	}
	return answer, nil
}

func (p *GiteaProvider) ListRepositories(org string) ([]*git.Repository, error) {
	answer := []*git.Repository{}
	if org == "" {
		repos, err := p.Client.ListMyRepos()
		if err != nil {
			return answer, err
		}
		for _, repo := range repos {
			answer = append(answer, toGiteaRepo(repo.Name, repo))
		}
		return answer, nil
	}
	repos, err := p.Client.ListOrgRepos(org)
	if err != nil {
		return answer, err
	}
	for _, repo := range repos {
		answer = append(answer, toGiteaRepo(repo.Name, repo))
	}
	return answer, nil
}

func (p *GiteaProvider) ListReleases(org string, name string) ([]*git.Release, error) {
	owner := org
	if owner == "" {
		owner = p.Username
	}
	answer := []*git.Release{}
	repos, err := p.Client.ListReleases(owner, name)
	if err != nil {
		return answer, err
	}
	for _, repo := range repos {
		answer = append(answer, toGiteaRelease(org, name, repo))
	}
	return answer, nil
}

func toGiteaRelease(org string, name string, release *gitea.Release) *git.Release {
	totalDownloadCount := 0
	assets := make([]git.ReleaseAsset, 0)
	for _, asset := range release.Attachments {
		totalDownloadCount = totalDownloadCount + int(asset.DownloadCount)
		assets = append(assets, git.ReleaseAsset{
			Name:               asset.Name,
			BrowserDownloadURL: asset.DownloadURL,
		})
	}
	return &git.Release{
		Name:          release.Title,
		TagName:       release.TagName,
		Body:          release.Note,
		URL:           release.URL,
		HTMLURL:       release.URL,
		DownloadCount: totalDownloadCount,
		Assets:        &assets,
	}
}

func (p *GiteaProvider) CreateRepository(org string, name string, private bool) (*git.Repository, error) {
	options := gitea.CreateRepoOption{
		Name:    name,
		Private: private,
	}
	repo, err := p.Client.CreateRepo(options)
	if err != nil {
		return nil, fmt.Errorf("Failed to create repository %s/%s due to: %s", org, name, err)
	}
	return toGiteaRepo(name, repo), nil
}

func (p *GiteaProvider) GetRepository(org string, name string) (*git.Repository, error) {
	repo, err := p.Client.GetRepo(org, name)
	if err != nil {
		return nil, fmt.Errorf("Failed to get repository %s/%s due to: %s", org, name, err)
	}
	return toGiteaRepo(name, repo), nil
}

func (p *GiteaProvider) DeleteRepository(org string, name string) error {
	owner := org
	if owner == "" {
		owner = p.Username
	}
	err := p.Client.DeleteRepo(owner, name)
	if err != nil {
		return fmt.Errorf("Failed to delete repository %s/%s due to: %s", owner, name, err)
	}
	return err
}

func toGiteaRepo(name string, repo *gitea.Repository) *git.Repository {
	return &git.Repository{
		Name:             name,
		AllowMergeCommit: true,
		CloneURL:         repo.CloneURL,
		HTMLURL:          repo.HTMLURL,
		SSHURL:           repo.SSHURL,
		Fork:             repo.Fork,
	}
}

func (p *GiteaProvider) ForkRepository(originalOrg string, name string, destinationOrg string) (*git.Repository, error) {
	repoConfig := gitea.CreateForkOption{
		Organization: &destinationOrg,
	}
	repo, err := p.Client.CreateFork(originalOrg, name, repoConfig)
	if err != nil {
		msg := ""
		if destinationOrg != "" {
			msg = fmt.Sprintf(" to %s", destinationOrg)
		}
		owner := destinationOrg
		if owner == "" {
			owner = p.Username
		}
		if strings.Contains(err.Error(), "try again later") {
			log.Warnf("Waiting for the fork of %s/%s to appear...\n", owner, name)
			// lets wait for the fork to occur...
			start := time.Now()
			deadline := start.Add(time.Minute)
			for {
				time.Sleep(5 * time.Second)
				repo, err = p.Client.GetRepo(owner, name)
				if repo != nil && err == nil {
					break
				}
				t := time.Now()
				if t.After(deadline) {
					return nil, fmt.Errorf("Gave up waiting for Repository %s/%s to appear: %s", owner, name, err)
				}
			}
		} else {
			return nil, fmt.Errorf("Failed to fork repository %s/%s%s due to: %s", originalOrg, name, msg, err)
		}
	}
	return toGiteaRepo(name, repo), nil
}

func (p *GiteaProvider) CreateWebHook(data *git.WebhookArguments) error {
	owner := data.Owner
	if owner == "" {
		owner = p.Username
	}
	repo := data.Repo.Name
	if repo == "" {
		return fmt.Errorf("Missing property Repo")
	}
	webhookUrl := data.URL
	if repo == "" {
		return fmt.Errorf("Missing property URL")
	}
	hooks, err := p.Client.ListRepoHooks(owner, repo)
	if err != nil {
		return err
	}
	for _, hook := range hooks {
		s := hook.Config["url"]
		if s == webhookUrl {
			log.Warnf("Already has a webhook registered for %s\n", webhookUrl)
			return nil
		}
	}
	config := map[string]string{
		"url":          webhookUrl,
		"content_type": "json",
	}
	if data.Secret != "" {
		config["secret"] = data.Secret
	}
	hook := gitea.CreateHookOption{
		Type:   "gitea",
		Config: config,
		Events: []string{"create", "push", "pull_request"},
		Active: true,
	}
	log.Infof("Creating Gitea webhook for %s/%s for url %s\n", util.ColorInfo(owner), util.ColorInfo(repo), util.ColorInfo(webhookUrl))
	_, err = p.Client.CreateRepoHook(owner, repo, hook)
	if err != nil {
		return fmt.Errorf("Failed to create webhook for %s/%s with %#v due to: %s", owner, repo, hook, err)
	}
	return err
}

func (p *GiteaProvider) ListWebHooks(owner string, repo string) ([]*git.WebhookArguments, error) {
	webHooks := []*git.WebhookArguments{}
	return webHooks, fmt.Errorf("not implemented!")
}

func (p *GiteaProvider) UpdateWebHook(data *git.WebhookArguments) error {
	return fmt.Errorf("not implemented!")
}

func (p *GiteaProvider) CreatePullRequest(data *git.PullRequestArguments) (*git.PullRequest, error) {
	owner := data.Repository.Organisation
	repo := data.Repository.Name
	title := data.Title
	body := data.Body
	head := data.Head
	base := data.Base
	config := gitea.CreatePullRequestOption{}
	if title != "" {
		config.Title = title
	}
	if body != "" {
		config.Body = body
	}
	if head != "" {
		config.Head = head
	}
	if base != "" {
		config.Base = base
	}
	pr, err := p.Client.CreatePullRequest(owner, repo, config)
	if err != nil {
		return nil, err
	}
	id := int(pr.Index)
	answer := &git.PullRequest{
		URL:    pr.HTMLURL,
		Number: &id,
		Owner:  data.Repository.Organisation,
		Repo:   data.Repository.Name,
	}
	if pr.Head != nil {
		answer.LastCommitSha = pr.Head.Sha
	}
	return answer, nil
}

func (p *GiteaProvider) UpdatePullRequestStatus(pr *git.PullRequest) error {
	if pr.Number == nil {
		return fmt.Errorf("Missing Number for PullRequest %#v", pr)
	}
	n := *pr.Number
	result, err := p.Client.GetPullRequest(pr.Owner, pr.Repo, int64(n))
	if err != nil {
		return fmt.Errorf("Could not find pull request for %s/%s #%d: %s", pr.Owner, pr.Repo, n, err)
	}
	pr.Author = &git.User{
		Login: result.Poster.UserName,
	}
	merged := result.HasMerged
	pr.Merged = &merged
	pr.Mergeable = &result.Mergeable
	pr.MergedAt = result.Merged
	pr.MergeCommitSHA = result.MergedCommitID
	pr.Title = result.Title
	pr.Body = result.Body
	stateText := string(result.State)
	pr.State = &stateText
	head := result.Head
	if head != nil {
		pr.LastCommitSha = head.Sha
	} else {
		pr.LastCommitSha = ""
	}
	/*
		TODO

		pr.ClosedAt = result.Closed
		pr.StatusesURL = result.StatusesURL
		pr.IssueURL = result.IssueURL
		pr.DiffURL = result.DiffURL
	*/
	return nil
}

func (p *GiteaProvider) GetPullRequest(owner string, repo *git.Repository, number int) (*git.PullRequest, error) {
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

func (p *GiteaProvider) GetPullRequestCommits(owner string, repository *git.Repository, number int) ([]*git.Commit, error) {
	answer := []*git.Commit{}

	// TODO there does not seem to be any way to get a diff of commits
	// unless maybe checking out the repo (do we have access to a local copy?)
	// there is a pr.Base and pr.Head that might be able to compare to get
	// commits somehow, but does not look like anything through the api

	return answer, nil
}

func (p *GiteaProvider) GetIssue(org string, name string, number int) (*git.Issue, error) {
	i, err := p.Client.GetIssue(org, name, int64(number))
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	return p.fromGiteaIssue(org, name, i)
}

func (p *GiteaProvider) IssueURL(org string, name string, number int, isPull bool) string {
	serverPrefix := p.ServerURL()
	if strings.Index(serverPrefix, "://") < 0 {
		serverPrefix = "https://" + serverPrefix
	}
	path := "issues"
	if isPull {
		path = "pull"
	}
	url := util.UrlJoin(serverPrefix, org, name, path, strconv.Itoa(number))
	return url
}

func (p *GiteaProvider) SearchIssues(org string, name string, filter string) ([]*git.Issue, error) {
	opts := gitea.ListIssueOption{}
	// TODO apply the filter?
	return p.searchIssuesWithOptions(org, name, opts)
}

func (p *GiteaProvider) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*git.Issue, error) {
	opts := gitea.ListIssueOption{}
	issues, err := p.searchIssuesWithOptions(org, name, opts)
	if err != nil {
		return issues, err
	}
	return git.FilterIssuesClosedSince(issues, t), nil
}

func (p *GiteaProvider) searchIssuesWithOptions(org string, name string, opts gitea.ListIssueOption) ([]*git.Issue, error) {
	opts.Page = 0
	answer := []*git.Issue{}
	issues, err := p.Client.ListRepoIssues(org, name, opts)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return answer, nil
		}
		return answer, err
	}
	for _, issue := range issues {
		i, err := p.fromGiteaIssue(org, name, issue)
		if err != nil {
			return answer, err
		}
		answer = append(answer, i)
	}
	return answer, nil
}

func (p *GiteaProvider) fromGiteaIssue(org string, name string, i *gitea.Issue) (*git.Issue, error) {
	state := string(i.State)
	labels := []git.Label{}
	for _, label := range i.Labels {
		labels = append(labels, toGiteaLabel(label))
	}
	assignees := []git.User{}
	assignee := i.Assignee
	if assignee != nil {
		assignees = append(assignees, *toGiteaUser(assignee))
	}
	number := int(i.ID)
	return &git.Issue{
		Number:        &number,
		URL:           p.IssueURL(org, name, number, false),
		State:         &state,
		Title:         i.Title,
		Body:          i.Body,
		IsPullRequest: i.PullRequest != nil,
		Labels:        labels,
		User:          toGiteaUser(i.Poster),
		Assignees:     assignees,
		CreatedAt:     &i.Created,
		UpdatedAt:     &i.Updated,
		ClosedAt:      i.Closed,
	}, nil
}

func (p *GiteaProvider) CreateIssue(owner string, repo string, issue *git.Issue) (*git.Issue, error) {
	config := gitea.CreateIssueOption{
		Title: issue.Title,
		Body:  issue.Body,
	}
	i, err := p.Client.CreateIssue(owner, repo, config)
	if err != nil {
		return nil, err
	}
	return p.fromGiteaIssue(owner, repo, i)
}

func toGiteaLabel(label *gitea.Label) git.Label {
	return git.Label{
		Name:  label.Name,
		Color: label.Color,
		URL:   label.URL,
	}
}

func toGiteaUser(user *gitea.User) *git.User {
	return &git.User{
		Login:     user.UserName,
		Name:      user.FullName,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
	}
}

func (p *GiteaProvider) MergePullRequest(pr *git.PullRequest, message string) error {
	if pr.Number == nil {
		return fmt.Errorf("Missing Number for PullRequest %#v", pr)
	}
	n := *pr.Number
	return p.Client.MergePullRequest(pr.Owner, pr.Repo, int64(n))
}

func (p *GiteaProvider) PullRequestLastCommitStatus(pr *git.PullRequest) (string, error) {
	ref := pr.LastCommitSha
	if ref == "" {
		return "", fmt.Errorf("Missing String for LastCommitSha %#v", pr)
	}
	results, err := p.Client.ListStatuses(pr.Owner, pr.Repo, ref, gitea.ListStatusesOption{})
	if err != nil {
		return "", err
	}
	for _, result := range results {
		text := string(result.State)
		if text != "" {
			return text, nil
		}
	}
	return "", fmt.Errorf("Could not find a status for repository %s/%s with ref %s", pr.Owner, pr.Repo, ref)
}

func (p *GiteaProvider) AddPRComment(pr *git.PullRequest, comment string) error {
	if pr.Number == nil {
		return fmt.Errorf("Missing Number for PullRequest %#v", pr)
	}
	n := *pr.Number
	prComment := gitea.CreateIssueCommentOption{
		Body: asText(&comment),
	}
	_, err := p.Client.CreateIssueComment(pr.Owner, pr.Repo, int64(n), prComment)
	return err
}

func (p *GiteaProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	issueComment := gitea.CreateIssueCommentOption{
		Body: comment,
	}
	_, err := p.Client.CreateIssueComment(owner, repo, int64(number), issueComment)
	if err != nil {
		return err
	}
	return nil
}

func (p *GiteaProvider) ListCommitStatus(org string, repo string, sha string) ([]*git.RepoStatus, error) {
	answer := []*git.RepoStatus{}
	results, err := p.Client.ListStatuses(org, repo, sha, gitea.ListStatusesOption{})
	if err != nil {
		return answer, fmt.Errorf("Could not find a status for repository %s/%s with ref %s", org, repo, sha)
	}
	for _, result := range results {
		status := &git.RepoStatus{
			ID:          string(result.ID),
			Context:     result.Context,
			URL:         result.URL,
			TargetURL:   result.TargetURL,
			State:       string(result.State),
			Description: result.Description,
		}
		answer = append(answer, status)
	}
	return answer, nil
}

func (b *GiteaProvider) UpdateCommitStatus(org string, repo string, sha string, status *git.RepoStatus) (*git.RepoStatus, error) {
	return &git.RepoStatus{}, errors.New("TODO")
}

func (p *GiteaProvider) RenameRepository(org string, name string, newName string) (*git.Repository, error) {
	return nil, fmt.Errorf("Rename of repositories is not supported for Gitea")
}

func (p *GiteaProvider) ValidateRepositoryName(org string, name string) error {
	_, err := p.Client.GetRepo(org, name)
	if err == nil {
		return fmt.Errorf("Repository %s already exists", p.Git.RepoName(org, name))
	}
	if strings.Contains(err.Error(), "404") {
		return nil
	}
	return err
}

func (p *GiteaProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *git.Release) error {
	var release *gitea.Release
	releases, err := p.Client.ListReleases(owner, repo)
	found := false
	for _, rel := range releases {
		if rel.TagName == tag {
			release = rel
			found = true
			break
		}
	}
	flag := false

	// lets populate the release
	if !found {
		createRelease := gitea.CreateReleaseOption{
			TagName:      releaseInfo.TagName,
			Title:        releaseInfo.Name,
			Note:         releaseInfo.Body,
			IsDraft:      flag,
			IsPrerelease: flag,
		}
		_, err = p.Client.CreateRelease(owner, repo, createRelease)
		return err
	} else {
		editRelease := gitea.EditReleaseOption{
			TagName:      release.TagName,
			Title:        release.Title,
			Note:         release.Note,
			IsDraft:      &flag,
			IsPrerelease: &flag,
		}
		if editRelease.Title == "" && releaseInfo.Name != "" {
			editRelease.Title = releaseInfo.Name
		}
		if editRelease.TagName == "" && releaseInfo.TagName != "" {
			editRelease.TagName = releaseInfo.TagName
		}
		if editRelease.Note == "" && releaseInfo.Body != "" {
			editRelease.Note = releaseInfo.Body
		}
		r2, err := p.Client.EditRelease(owner, repo, release.ID, editRelease)
		if err != nil {
			return err
		}
		if r2 != nil {
			releaseInfo.URL = r2.URL
		}
	}
	return err
}

func (p *GiteaProvider) HasIssues() bool {
	return true
}

func (p *GiteaProvider) IsGitHub() bool {
	return false
}

func (p *GiteaProvider) IsGitea() bool {
	return true
}

func (p *GiteaProvider) IsBitbucketCloud() bool {
	return false
}

func (p *GiteaProvider) IsBitbucketServer() bool {
	return false
}

func (p *GiteaProvider) IsGerrit() bool {
	return false
}

func (p *GiteaProvider) Kind() string {
	return "gitea"
}

func (p *GiteaProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/gitea-webhook/post"
}

func (p *GiteaProvider) AccessTokenURL() string {
	return util.UrlJoin(p.ServerURL(), "/user/settings/applications")
}

func (p *GiteaProvider) Label() string {
	return p.Name
}

func (p *GiteaProvider) ServerURL() string {
	return p.URL
}

func (p *GiteaProvider) BranchArchiveURL(org string, name string, branch string) string {
	return util.UrlJoin(p.ServerURL(), org, name, "archive", branch+".zip")
}

func (p *GiteaProvider) CurrentUsername() string {
	return p.Username
}

func (p *GiteaProvider) UserInfo(username string) *git.User {
	user, err := p.Client.GetUserInfo(username)

	if err != nil {
		return nil
	}

	return &git.User{
		Login:     username,
		Name:      user.FullName,
		AvatarURL: user.AvatarURL,
		Email:     user.Email,
		// TODO figure the Gitea user url
		URL: p.URL + "/" + username,
	}
}

func (p *GiteaProvider) AddCollaborator(user string, organisation string, repo string) error {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for Gitea. Please add user: %v as a collaborator to this project.\n", user)
	return nil
}

func (p *GiteaProvider) ListInvitations() ([]*github.RepositoryInvitation, *github.Response, error) {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for Gitea.\n")
	return []*github.RepositoryInvitation{}, &github.Response{}, nil
}

func (p *GiteaProvider) AcceptInvitation(ID int64) (*github.Response, error) {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for Gitea.\n")
	return &github.Response{}, nil
}

func (p *GiteaProvider) GetContent(org string, name string, path string, ref string) (*git.FileContent, error) {
	return nil, fmt.Errorf("Getting content not supported on gitea")
}

func asText(text *string) string {
	if text != nil {
		return *text
	}
	return ""
}
