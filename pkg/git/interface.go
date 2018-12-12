package git

import (
	"time"

	"github.com/google/go-github/github"
	"github.com/jenkins-x/jx/pkg/auth"
)

// OrganisationLister returns a slice of GitOrganisation
//go:generate pegomock generate github.com/wbrefvem/go-gits/pkg/git OrganisationLister -o mocks/organisation_lister.go --generate-matchers
type OrganisationLister interface {
	ListOrganisations() ([]GitOrganisation, error)
}

// OrganisationChecker verifies if an user is member of an organization
//go:generate pegomock generate github.com/wbrefvem/go-gits/pkg/git OrganisationChecker -o mocks/organisation_checker.go --generate-matchers
type OrganisationChecker interface {
	IsUserInOrganisation(user string, organisation string) (bool, error)
}

// GitProvider is the interface for abstracting use of different git provider APIs
//go:generate pegomock generate github.com/wbrefvem/go-gits/pkg/git GitProvider -o mocks/git_provider.go --generate-matchers
type GitProvider interface {
	OrganisationLister

	ListRepositories(org string) ([]*GitRepository, error)

	CreateRepository(org string, name string, private bool) (*GitRepository, error)

	GetRepository(org string, name string) (*GitRepository, error)

	DeleteRepository(org string, name string) error

	ForkRepository(originalOrg string, name string, destinationOrg string) (*GitRepository, error)

	RenameRepository(org string, name string, newName string) (*GitRepository, error)

	ValidateRepositoryName(org string, name string) error

	CreatePullRequest(data *GitPullRequestArguments) (*GitPullRequest, error)

	UpdatePullRequestStatus(pr *GitPullRequest) error

	GetPullRequest(owner string, repo *GitRepository, number int) (*GitPullRequest, error)

	GetPullRequestCommits(owner string, repo *GitRepository, number int) ([]*GitCommit, error)

	PullRequestLastCommitStatus(pr *GitPullRequest) (string, error)

	ListCommitStatus(org string, repo string, sha string) ([]*GitRepoStatus, error)

	UpdateCommitStatus(org string, repo string, sha string, status *GitRepoStatus) (*GitRepoStatus, error)

	MergePullRequest(pr *GitPullRequest, message string) error

	CreateWebHook(data *GitWebHookArguments) error

	ListWebHooks(org string, repo string) ([]*GitWebHookArguments, error)

	UpdateWebHook(data *GitWebHookArguments) error

	IsGitHub() bool

	IsGitea() bool

	IsBitbucketCloud() bool

	IsBitbucketServer() bool

	IsGerrit() bool

	Kind() string

	GetIssue(org string, name string, number int) (*GitIssue, error)

	IssueURL(org string, name string, number int, isPull bool) string

	SearchIssues(org string, name string, query string) ([]*GitIssue, error)

	SearchIssuesClosedSince(org string, name string, t time.Time) ([]*GitIssue, error)

	CreateIssue(owner string, repo string, issue *GitIssue) (*GitIssue, error)

	HasIssues() bool

	AddPRComment(pr *GitPullRequest, comment string) error

	CreateIssueComment(owner string, repo string, number int, comment string) error

	UpdateRelease(owner string, repo string, tag string, releaseInfo *GitRelease) error

	ListReleases(org string, name string) ([]*GitRelease, error)

	GetContent(org string, name string, path string, ref string) (*GitFileContent, error)

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
	// * gitea
	// /gitea-webhook/post
	//
	// * generic webhook
	// /generic-webhook-trigger/invoke?token=abc123
	// https://wiki.jenkins.io/display/JENKINS/Generic+Webhook+Trigger+Plugin

	JenkinsWebHookPath(gitURL string, secret string) string

	// Label returns the Git service label or name
	Label() string

	// ServerURL returns the Git server URL
	ServerURL() string

	// BranchArchiveURL returns a URL to the ZIP archive for the git branch
	BranchArchiveURL(org string, name string, branch string) string

	// Returns the current username
	CurrentUsername() string

	// Returns the current user auth
	UserAuth() auth.UserAuth

	// Returns user info, if possible
	UserInfo(username string) *GitUser

	AddCollaborator(string, string, string) error
	// TODO Refactor to remove bespoke types when we implement another provider
	ListInvitations() ([]*github.RepositoryInvitation, *github.Response, error)
	// TODO Refactor to remove bespoke types when we implement another provider
	AcceptInvitation(int64) (*github.Response, error)
}
