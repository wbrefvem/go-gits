package bitbucketcloud

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/util"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/wbrefvem/go-bitbucket"
	"github.com/wbrefvem/go-gits/pkg/git"
)

// CloudProvider implements git.Provider interface for bitbucket.org
type CloudProvider struct {
	Client   *bitbucket.APIClient
	Username string
	Context  context.Context
	URL      string
	Name     string
	Git      git.Gitter
}

var stateMap = map[string]string{
	"SUCCESSFUL": "success",
	"FAILED":     "failure",
	"INPROGRESS": "in-progress",
	"STOPPED":    "stopped",
}

func NewBitbucketCloudProvider(username, serverURL, token, providerName string, git git.Gitter) (git.Provider, error) {
	ctx := context.Background()

	basicAuth := bitbucket.BasicAuth{
		UserName: username,
		Password: token,
	}
	basicAuthContext := context.WithValue(ctx, bitbucket.ContextBasicAuth, basicAuth)

	provider := CloudProvider{
		URL:      serverURL,
		Name:     providerName,
		Username: username,
		Context:  basicAuthContext,
		Git:      git,
	}

	cfg := bitbucket.NewConfiguration()
	provider.Client = bitbucket.NewAPIClient(cfg)

	return &provider, nil
}

func (b *CloudProvider) ListOrganisations() ([]git.Organisation, error) {

	teams := []git.Organisation{}

	var results bitbucket.PaginatedTeams
	var err error

	// Pagination is gross.
	for {
		if results.Next == "" {
			results, _, err = b.Client.TeamsApi.TeamsGet(b.Context, map[string]interface{}{"role": "member"})
		} else {
			results, _, err = b.Client.PagingApi.TeamsPageGet(b.Context, results.Next)
		}

		if err != nil {
			return nil, err
		}

		for _, team := range results.Values {
			teams = append(teams, git.Organisation{Login: team.Username})
		}

		if results.Next == "" {
			break
		}
	}

	return teams, nil
}

func BitbucketRepositoryToGitRepository(bRepo bitbucket.Repository) *git.Repository {
	var sshURL string
	var httpCloneURL string
	for _, link := range bRepo.Links.Clone {
		if link.Name == "ssh" {
			sshURL = link.Href
		}
	}
	isFork := false
	if bRepo.Parent != nil {
		isFork = true
	}
	if httpCloneURL == "" {
		httpCloneURL = bRepo.Links.Html.Href
		if !strings.HasSuffix(httpCloneURL, ".git") {
			httpCloneURL += ".git"
		}
	}
	if httpCloneURL == "" {
		httpCloneURL = sshURL
	}
	return &git.Repository{
		Name:     bRepo.Name,
		HTMLURL:  bRepo.Links.Html.Href,
		CloneURL: httpCloneURL,
		SSHURL:   sshURL,
		Language: bRepo.Language,
		Fork:     isFork,
	}
}

func (b *CloudProvider) ListRepositories(org string) ([]*git.Repository, error) {

	repos := []*git.Repository{}

	var results bitbucket.PaginatedRepositories
	var err error

	for {
		if results.Next == "" {
			results, _, err = b.Client.RepositoriesApi.RepositoriesUsernameGet(b.Context, org, nil)
		} else {
			results, _, err = b.Client.PagingApi.RepositoriesPageGet(b.Context, results.Next)
		}

		if err != nil {
			return nil, err
		}

		for _, repo := range results.Values {
			repos = append(repos, BitbucketRepositoryToGitRepository(repo))
		}

		if results.Next == "" {
			break
		}
	}

	return repos, nil
}

func (b *CloudProvider) CreateRepository(
	org string,
	name string,
	private bool,
) (*git.Repository, error) {

	options := map[string]interface{}{}
	options["body"] = bitbucket.Repository{
		IsPrivate: private,
		Scm:       "git",
	}

	result, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugPost(
		b.Context,
		org,
		name,
		options,
	)

	if err != nil {
		return nil, err
	}

	return BitbucketRepositoryToGitRepository(result), nil
}

func (b *CloudProvider) GetRepository(
	org string,
	name string,
) (*git.Repository, error) {

	repo, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugGet(
		b.Context,
		org,
		name,
	)

	if err != nil {
		return nil, err
	}

	return BitbucketRepositoryToGitRepository(repo), nil
}

func (b *CloudProvider) DeleteRepository(org string, name string) error {

	_, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugDelete(
		b.Context,
		org,
		name,
		nil,
	)

	if err != nil {
		return err
	}

	return nil
}

func (b *CloudProvider) ForkRepository(
	originalOrg string,
	name string,
	destinationOrg string,
) (*git.Repository, error) {
	options := map[string]interface{}{
		"body": map[string]interface{}{},
	}
	repo, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugForksPost(
		b.Context,
		originalOrg,
		name,
		options,
	)

	if err != nil {
		return nil, err
	}

	_, _, err = b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugForksGet(
		b.Context,
		originalOrg,
		repo.Name,
	)

	// Fork isn't ready
	if err != nil {

		// Wait up to 1 minute for the fork to be ready
		for i := 0; i < 30; i++ {
			_, _, err = b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugForksGet(
				b.Context,
				originalOrg,
				repo.Name,
			)

			if err == nil {
				break
			}

			time.Sleep(2 * time.Second)
		}
	}

	return BitbucketRepositoryToGitRepository(repo), nil
}

func (b *CloudProvider) RenameRepository(
	org string,
	name string,
	newName string,
) (*git.Repository, error) {

	var options = map[string]interface{}{
		"name": newName,
	}

	repo, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugPut(
		b.Context,
		org,
		name,
		options,
	)

	if err != nil {
		return nil, err
	}

	return BitbucketRepositoryToGitRepository(repo), nil
}

func (b *CloudProvider) ValidateRepositoryName(org string, name string) error {

	_, r, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugGet(
		b.Context,
		org,
		name,
	)

	if r != nil && r.StatusCode == 404 {
		return nil
	}

	if err == nil {
		return fmt.Errorf("repository %s/%s already exists", org, name)
	}

	return err
}

func (b *CloudProvider) CreatePullRequest(
	data *git.PullRequestArguments,
) (*git.PullRequest, error) {

	head := bitbucket.PullrequestEndpointBranch{Name: data.Head}
	sourceFullName := fmt.Sprintf("%s/%s", data.Repository.Organisation, data.Repository.Name)
	sourceRepo := bitbucket.Repository{FullName: sourceFullName}
	source := bitbucket.PullrequestEndpoint{
		Repository: &sourceRepo,
		Branch:     &head,
	}

	base := bitbucket.PullrequestEndpointBranch{Name: data.Base}
	destination := bitbucket.PullrequestEndpoint{
		Branch: &base,
	}

	bPullrequest := bitbucket.Pullrequest{
		Source:      &source,
		Destination: &destination,
		Title:       data.Title,
	}

	var options = map[string]interface{}{
		"body": bPullrequest,
	}

	pr, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPost(
		b.Context,
		data.Repository.Organisation,
		data.Repository.Name,
		options,
	)

	if err != nil {
		return nil, err
	}

	_, _, err = b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet(
		b.Context,
		data.Repository.Organisation,
		data.Repository.Name,
		pr.Id,
	)

	if err != nil {
		// Wait up to 1 minute for the PR to be ready.
		for i := 0; i < 30; i++ {
			_, _, err = b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet(
				b.Context,
				data.Repository.Organisation,
				data.Repository.Name,
				pr.Id,
			)

			if err == nil {
				break
			}

			time.Sleep(2 * time.Second)
		}
	}

	i := int(pr.Id)
	prID := &i

	newPR := &git.PullRequest{
		URL:    pr.Links.Html.Href,
		Author: b.UserInfo(pr.Author.Username),
		Owner:  strings.Split(pr.Destination.Repository.FullName, "/")[0],
		Repo:   pr.Destination.Repository.Name,
		Number: prID,
		State:  &pr.State,
	}

	return newPR, nil
}

func (b *CloudProvider) UpdatePullRequestStatus(pr *git.PullRequest) error {

	prID := int32(*pr.Number)
	bitbucketPR, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet(
		b.Context,
		pr.Owner,
		pr.Repo,
		prID,
	)

	if err != nil {
		return err
	}

	pr.State = &bitbucketPR.State
	pr.Title = bitbucketPR.Title
	pr.Body = bitbucketPR.Summary.Raw
	pr.Author = b.UserInfo(bitbucketPR.Author.Username)

	if bitbucketPR.MergeCommit != nil {
		pr.MergeCommitSHA = &bitbucketPR.MergeCommit.Hash
	}
	pr.DiffURL = &bitbucketPR.Links.Diff.Href

	if bitbucketPR.State == "MERGED" {
		merged := true
		pr.Merged = &merged
	}

	commits, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommitsGet(
		b.Context,
		pr.Owner,
		strconv.FormatInt(int64(prID), 10),
		pr.Repo,
	)

	if err != nil {
		return err
	}

	values := commits["values"].([]interface{})
	commit := values[0].(map[string]interface{})

	pr.LastCommitSha = commit["hash"].(string)

	return nil
}

func (p *CloudProvider) GetPullRequest(owner string, repoInfo *git.Repository, number int) (*git.PullRequest, error) {
	repo := repoInfo.Name
	pr, _, err := p.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdGet(
		p.Context,
		owner,
		repo,
		int32(number),
	)

	if err != nil {
		return nil, err
	}

	author := p.UserInfo(pr.Author.Username)

	if author.Email == "" {
		// bitbucket makes this part difficult, there is no way to directly
		// associate a username to an email through the API or vice versa
		// so our best attempt is to try to figure out the author email
		// from the commits
		commits, err := p.GetPullRequestCommits(owner, repoInfo, number)

		if err != nil {
			log.Warn("Unable to get commits for PR: " + owner + "/" + repo + "/" + strconv.Itoa(number) + " -- " + err.Error())
		}

		// we get correct login and email per commit, find the matching author
		for _, commit := range commits {
			if commit.Author.Login == author.Login {
				author.Email = commit.Author.Email
				break
			}
		}
	}

	return &git.PullRequest{
		URL:    pr.Links.Html.Href,
		Owner:  strings.Split(pr.Destination.Repository.FullName, "/")[0],
		Repo:   pr.Destination.Repository.Name,
		Number: &number,
		State:  &pr.State,
		Author: author,
	}, nil
}

func (b *CloudProvider) GetPullRequestCommits(owner string, repository *git.Repository, number int) ([]*git.Commit, error) {
	repo := repository.Name
	answer := []*git.Commit{}

	// for some reason the 2nd parameter is the PR id, seems like an inconsistency/bug in the api
	commits, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdCommitsGet(b.Context, owner, strconv.Itoa(number), repo)
	if err != nil {
		return answer, err
	}

	commitVals, ok := commits["values"]
	if !ok {
		return answer, fmt.Errorf("No value key for %s/%s/%d", owner, repo, number)
	}

	commitValues, ok := commitVals.([]interface{})
	if !ok {
		return answer, fmt.Errorf("No commitValues for %s/%s/%d", owner, repo, number)
	}

	rawEmailMatcher, _ := regexp.Compile("[^<]*<([^>]+)>")

	for _, data := range commitValues {
		if data == nil {
			continue
		}

		comm, ok := data.(map[string]interface{})
		if !ok {
			log.Warn(fmt.Sprintf("Unexpected data structure for GetPullRequestCommits values from PR %s/%s/%d", owner, repo, number))
			continue
		}

		shaVal, ok := comm["hash"]
		if !ok {
			continue
		}

		sha, ok := shaVal.(string)
		if !ok {
			log.Warn(fmt.Sprintf("Unexpected data structure for GetPullRequestCommits hash from PR %s/%s/%d", owner, repo, number))
			continue
		}

		commit, _, err := b.Client.CommitsApi.RepositoriesUsernameRepoSlugCommitRevisionGet(b.Context, owner, repo, sha)
		if err != nil {
			return answer, err
		}

		url := ""
		if commit.Links != nil && commit.Links.Self != nil {
			url = commit.Links.Self.Href
		}

		// update the login and email
		login := ""
		email := ""
		if commit.Author != nil {
			// commit.Author is the actual Bitbucket user
			if commit.Author.User != nil {
				login = commit.Author.User.Username
			}
			// Author.Raw contains the Git commit author in the form: User <email@example.com>
			email = rawEmailMatcher.ReplaceAllString(commit.Author.Raw, "$1")
		}

		summary := &git.Commit{
			Message: commit.Message,
			URL:     url,
			SHA:     commit.Hash,
			Author: &git.User{
				Login: login,
				Email: email,
			},
		}

		answer = append(answer, summary)
	}
	return answer, nil
}

func (b *CloudProvider) PullRequestLastCommitStatus(pr *git.PullRequest) (string, error) {

	latestCommitStatus := bitbucket.Commitstatus{}

	var result bitbucket.PaginatedCommitstatuses
	var err error

	for {
		if result.Next == "" {
			result, _, err = b.Client.CommitstatusesApi.RepositoriesUsernameRepoSlugCommitNodeStatusesGet(
				b.Context,
				pr.Owner,
				pr.Repo,
				pr.LastCommitSha,
			)
		} else {
			result, _, err = b.Client.PagingApi.CommitstatusesPageGet(b.Context, result.Next)
		}

		if err != nil {
			return "", err
		}

		// Our first time building, so return "success"
		if result.Size == 0 {
			return "success", nil
		}

		for _, status := range result.Values {
			if status.CreatedOn.After(latestCommitStatus.CreatedOn) {
				latestCommitStatus = status
			}
		}

		if result.Next == "" {
			break
		}
	}

	return stateMap[latestCommitStatus.State], nil
}

func (b *CloudProvider) ListCommitStatus(org string, repo string, sha string) ([]*git.RepoStatus, error) {

	statuses := []*git.RepoStatus{}

	var result bitbucket.PaginatedCommitstatuses
	var err error

	for {
		if result.Next == "" {
			result, _, err = b.Client.CommitstatusesApi.RepositoriesUsernameRepoSlugCommitNodeStatusesGet(
				b.Context,
				org,
				repo,
				sha,
			)
		} else {
			result, _, err = b.Client.PagingApi.CommitstatusesPageGet(b.Context, result.Next)
		}

		if err != nil {
			return nil, err
		}

		for _, status := range result.Values {

			if err != nil {
				return nil, err
			}

			newStatus := &git.RepoStatus{
				ID:          status.Key,
				URL:         status.Links.Commit.Href,
				State:       stateMap[status.State],
				TargetURL:   status.Links.Self.Href,
				Description: status.Description,
			}
			statuses = append(statuses, newStatus)
		}

		if result.Next == "" {
			break
		}
	}
	return statuses, nil
}

func (b *CloudProvider) UpdateCommitStatus(org string, repo string, sha string, status *git.RepoStatus) (*git.RepoStatus, error) {
	return &git.RepoStatus{}, errors.New("TODO")
}

func (b *CloudProvider) MergePullRequest(pr *git.PullRequest, message string) error {

	options := map[string]interface{}{
		"body": map[string]interface{}{
			"pullrequest_merge_parameters": map[string]interface{}{
				"message": message,
			},
		},
	}

	_, _, err := b.Client.PullrequestsApi.RepositoriesUsernameRepoSlugPullrequestsPullRequestIdMergePost(
		b.Context,
		pr.Owner,
		strconv.FormatInt(int64(*pr.Number), 10),
		pr.Repo,
		options,
	)

	if err != nil {
		return err
	}

	return nil
}

func (b *CloudProvider) CreateWebHook(data *git.WebhookArguments) error {

	options := map[string]interface{}{
		"body": map[string]interface{}{
			"url":    data.URL,
			"active": true,
			"events": []string{
				"repo:push",
			},
			"description": "Jenkins X Web Hook",
		},
	}

	_, _, err := b.Client.RepositoriesApi.RepositoriesUsernameRepoSlugHooksPost(
		b.Context,
		data.Repo.Organisation,
		data.Repo.Name,
		options,
	)

	if err != nil {
		return err
	}
	return nil
}

func (p *CloudProvider) ListWebHooks(owner string, repo string) ([]*git.WebhookArguments, error) {
	webHooks := []*git.WebhookArguments{}
	return webHooks, fmt.Errorf("not implemented!")
}

func (p *CloudProvider) UpdateWebHook(data *git.WebhookArguments) error {
	return fmt.Errorf("not implemented!")
}

func BitbucketIssueToIssue(bIssue bitbucket.Issue) *git.Issue {
	id := int(bIssue.Id)
	ownerAndRepo := strings.Split(bIssue.Repository.FullName, "/")
	owner := ownerAndRepo[0]

	var assignee git.User

	if bIssue.Assignee != nil {
		assignee = git.User{
			URL:   bIssue.Assignee.Links.Self.Href,
			Login: bIssue.Assignee.Username,
			Name:  bIssue.Assignee.DisplayName,
		}
	}
	gitIssue := &git.Issue{
		URL:       bIssue.Links.Self.Href,
		Owner:     owner,
		Repo:      bIssue.Repository.Name,
		Number:    &id,
		Title:     bIssue.Title,
		Body:      bIssue.Content.Markup,
		State:     &bIssue.State,
		IssueURL:  &bIssue.Links.Html.Href,
		CreatedAt: &bIssue.CreatedOn,
		UpdatedAt: &bIssue.UpdatedOn,
		ClosedAt:  &bIssue.UpdatedOn,
		Assignees: []git.User{
			assignee,
		},
	}
	return gitIssue
}

func (b *CloudProvider) IssueToBitbucketIssue(gIssue git.Issue) bitbucket.Issue {

	bitbucketIssue := bitbucket.Issue{
		Title:      gIssue.Title,
		Repository: &bitbucket.Repository{Name: gIssue.Repo},
		Reporter:   &bitbucket.User{Username: b.Username},
	}

	return bitbucketIssue
}

func (b *CloudProvider) SearchIssues(org string, name string, query string) ([]*git.Issue, error) {

	gitIssues := []*git.Issue{}

	var issues bitbucket.PaginatedIssues
	var err error

	for {
		if issues.Next == "" {
			issues, _, err = b.Client.IssueTrackerApi.RepositoriesUsernameRepoSlugIssuesGet(b.Context, org, name)
		} else {
			issues, _, err = b.Client.PagingApi.IssuesPageGet(b.Context, issues.Next)
		}

		if err != nil {
			return nil, err
		}

		for _, issue := range issues.Values {
			gitIssues = append(gitIssues, BitbucketIssueToIssue(issue))
		}

		if issues.Next == "" {
			break
		}
	}

	return gitIssues, nil
}

func (b *CloudProvider) SearchIssuesClosedSince(org string, name string, t time.Time) ([]*git.Issue, error) {
	issues, err := b.SearchIssues(org, name, "")
	if err != nil {
		return issues, err
	}
	return git.FilterIssuesClosedSince(issues, t), nil
}

func (b *CloudProvider) GetIssue(org string, name string, number int) (*git.Issue, error) {

	issue, _, err := b.Client.IssueTrackerApi.RepositoriesUsernameRepoSlugIssuesIssueIdGet(
		b.Context,
		org,
		strconv.FormatInt(int64(number), 10),
		name,
	)

	if err != nil {
		return nil, err
	}
	return BitbucketIssueToIssue(issue), nil
}

func (p *CloudProvider) IssueURL(org string, name string, number int, isPull bool) string {
	serverPrefix := p.URL
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

func (b *CloudProvider) CreateIssue(owner string, repo string, issue *git.Issue) (*git.Issue, error) {

	bIssue, _, err := b.Client.IssueTrackerApi.RepositoriesUsernameRepoSlugIssuesPost(
		b.Context,
		owner,
		repo,
		b.IssueToBitbucketIssue(*issue),
	)

	// We need to make a second round trip to get the issue's HTML URL.
	bIssue, _, err = b.Client.IssueTrackerApi.RepositoriesUsernameRepoSlugIssuesIssueIdGet(
		b.Context,
		owner,
		strconv.FormatInt(int64(bIssue.Id), 10),
		repo,
	)

	if err != nil {
		return nil, err
	}
	return BitbucketIssueToIssue(bIssue), nil
}

func (b *CloudProvider) AddPRComment(pr *git.PullRequest, comment string) error {
	log.Warn("Bitbucket Cloud doesn't support adding PR comments via the REST API")
	return nil
}

func (b *CloudProvider) CreateIssueComment(owner string, repo string, number int, comment string) error {
	log.Warn("Bitbucket Cloud doesn't support adding issue comments viea the REST API")
	return nil
}

func (b *CloudProvider) HasIssues() bool {
	return true
}

func (b *CloudProvider) IsGitHub() bool {
	return false
}

func (b *CloudProvider) IsGitea() bool {
	return false
}

func (b *CloudProvider) IsBitbucketCloud() bool {
	return true
}

func (b *CloudProvider) IsBitbucketServer() bool {
	return false
}

func (b *CloudProvider) IsGerrit() bool {
	return false
}

func (b *CloudProvider) Kind() string {
	return "bitbucketcloud"
}

// Exposed by Jenkins plugin; this one is for https://wiki.jenkins.io/display/JENKINS/BitBucket+Plugin
func (b *CloudProvider) JenkinsWebHookPath(gitURL string, secret string) string {
	return "/bitbucket-scmsource-hook/notify"
}

func (b *CloudProvider) Label() string {
	return b.Name
}

func (b *CloudProvider) ServerURL() string {
	return b.URL
}

func (b *CloudProvider) BranchArchiveURL(org string, name string, branch string) string {
	return util.UrlJoin(b.ServerURL(), org, name, "get", branch+".zip")
}

func (p *CloudProvider) CurrentUsername() string {
	return p.Username
}

func (p *CloudProvider) UserInfo(username string) *git.User {
	user, _, err := p.Client.UsersApi.UsersUsernameGet(p.Context, username)
	if err != nil {
		log.Error("Unable to fetch user info for " + username + " due to " + err.Error() + "\n")
		return nil
	}

	return &git.User{
		Login:     username,
		Name:      user.DisplayName,
		AvatarURL: user.Links.Avatar.Href,
		URL:       user.Links.Self.Href,
	}
}

func (b *CloudProvider) UpdateRelease(owner string, repo string, tag string, releaseInfo *git.Release) error {
	log.Warn("Bitbucket Cloud doesn't support releases")
	return nil
}

func (p *CloudProvider) ListReleases(org string, name string) ([]*git.Release, error) {
	answer := []*git.Release{}
	log.Warn("Bitbucket Cloud doesn't support releases")
	return answer, nil
}

func (b *CloudProvider) AddCollaborator(user string, organisation string, repo string) error {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for bitbucket. Please add user: %v as a collaborator to this project.\n", user)
	return nil
}

func (b *CloudProvider) ListInvitations() ([]*github.RepositoryInvitation, *github.Response, error) {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for bitbucket.\n")
	return []*github.RepositoryInvitation{}, &github.Response{}, nil
}

func (b *CloudProvider) AcceptInvitation(ID int64) (*github.Response, error) {
	log.Infof("Automatically adding the pipeline user as a collaborator is currently not implemented for bitbucket.\n")
	return &github.Response{}, nil
}

func (b *CloudProvider) GetContent(org string, name string, path string, ref string) (*git.FileContent, error) {
	return nil, fmt.Errorf("Getting content not supported on bitbucket")
}

func (b *CloudProvider) AccessTokenURL() string {
	// TODO with github we can default the scopes/flags we need on a token via adding
	// ?scopes=repo,read:user,user:email,write:repo_hook
	//
	// is there a way to do that for bitbucket?
	return util.UrlJoin(b.ServerURL(), "/account/user", b.Username, "/app-passwords/new")
}
