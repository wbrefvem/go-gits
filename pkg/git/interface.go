package git

import (
	"time"

	gitcfg "gopkg.in/src-d/go-git.v4/config"
)

// OrganisationLister returns a slice of Organisation
//go:generate pegomock generate github.com/wbrefvem/go-gits/pkg/git OrganisationLister -o mocks/organisation_lister.go --generate-matchers
type OrganisationLister interface {
	ListOrganisations() ([]Organisation, error)
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

	ListRepositories(org string) ([]*Repository, error)

	CreateRepository(org string, name string, private bool) (*Repository, error)

	GetRepository(org string, name string) (*Repository, error)

	DeleteRepository(org string, name string) error

	ForkRepository(originalOrg string, name string, destinationOrg string) (*Repository, error)

	RenameRepository(org string, name string, newName string) (*Repository, error)

	ValidateRepositoryName(org string, name string) error

	CreatePullRequest(data *PullRequestArguments) (*PullRequest, error)

	UpdatePullRequestStatus(pr *PullRequest) error

	GetPullRequest(owner string, repo *Repository, number int) (*PullRequest, error)

	GetPullRequestCommits(owner string, repo *Repository, number int) ([]*Commit, error)

	PullRequestLastCommitStatus(pr *PullRequest) (string, error)

	ListCommitStatus(org string, repo string, sha string) ([]*RepoStatus, error)

	UpdateCommitStatus(org string, repo string, sha string, status *RepoStatus) (*RepoStatus, error)

	MergePullRequest(pr *PullRequest, message string) error

	CreateWebHook(data *WebhookArguments) error

	ListWebHooks(org string, repo string) ([]*WebhookArguments, error)

	UpdateWebHook(data *WebhookArguments) error

	IsGitHub() bool

	IsGitea() bool

	IsBitbucketCloud() bool

	IsBitbucketServer() bool

	IsGerrit() bool

	Kind() string

	GetIssue(org string, name string, number int) (*Issue, error)

	IssueURL(org string, name string, number int, isPull bool) string

	SearchIssues(org string, name string, query string) ([]*Issue, error)

	SearchIssuesClosedSince(org string, name string, t time.Time) ([]*Issue, error)

	CreateIssue(owner string, repo string, issue *Issue) (*Issue, error)

	HasIssues() bool

	AddPRComment(pr *PullRequest, comment string) error

	CreateIssueComment(owner string, repo string, number int, comment string) error

	UpdateRelease(owner string, repo string, tag string, releaseInfo *Release) error

	ListReleases(org string, name string) ([]*Release, error)

	GetContent(org string, name string, path string, ref string) (*FileContent, error)

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

	// Returns user info, if possible
	UserInfo(username string) *User

	AccessTokenURL() string
}

// Gitter defines common git actions used by Jenkins X via git cli
//go:generate pegomock generate github.com/wbrefvem/go-gits/pkg/git Gitter -o mocks/gitter.go --generate-matchers
type Gitter interface {
	FindGitConfigDir(dir string) (string, string, error)

	Status(dir string) error
	Server(dir string) (string, error)
	Info(dir string) (*Repository, error)
	IsFork(dir string) (bool, error)
	Version() (string, error)
	RepoName(org, repoName string) string

	Username(dir string) (string, error)
	SetUsername(dir string, username string) error
	Email(dir string) (string, error)
	SetEmail(dir string, email string) error
	GetAuthorEmailForCommit(dir string, sha string) (string, error)

	Init(dir string) error
	Clone(url string, directory string) error
	ShallowCloneBranch(url string, branch string, directory string) error
	Push(dir string) error
	PushMaster(dir string) error
	PushTag(dir string, tag string) error
	CreatePushURL(cloneURL, username, token string) (string, error)
	ForcePushBranch(dir string, localBranch string, remoteBranch string) error
	CloneOrPull(url string, directory string) error
	Pull(dir string) error
	PullRemoteBranches(dir string) error
	PullUpstream(dir string) error

	AddRemote(dir string, name string, url string) error
	SetRemoteURL(dir string, name string, gitURL string) error
	UpdateRemote(dir, url string) error
	DiscoverRemoteGitURL(gitConf string) (string, error)
	DiscoverUpstreamGitURL(gitConf string) (string, error)
	RemoteBranches(dir string) ([]string, error)
	RemoteBranchNames(dir string, prefix string) ([]string, error)
	GetRemoteUrl(config *gitcfg.Config, name string) string

	Branch(dir string) (string, error)
	CreateBranch(dir string, branch string) error
	CheckoutRemoteBranch(dir string, branch string) error
	Checkout(dir string, branch string) error
	CheckoutOrphan(dir string, branch string) error
	ConvertToValidBranchName(name string) string
	FetchBranch(dir string, repo string, refspec string) error

	Stash(dir string) error

	Remove(dir, fileName string) error
	RemoveForce(dir, fileName string) error
	CleanForce(dir, fileName string) error
	Add(dir string, args ...string) error

	CommitIfChanges(dir string, message string) error
	CommitDir(dir string, message string) error
	AddCommit(dir string, msg string) error
	HasChanges(dir string) (bool, error)
	Diff(dir string) (string, error)

	GetLatestCommitMessage(dir string) (string, error)
	GetPreviousGitTagSHA(dir string) (string, error)
	GetCurrentGitTagSHA(dir string) (string, error)
	FetchTags(dir string) error
	Tags(dir string) ([]string, error)
	CreateTag(dir string, tag string, msg string) error

	GetRevisionBeforeDate(dir string, t time.Time) (string, error)
	GetRevisionBeforeDateText(dir string, dateText string) (string, error)
	DeleteRemoteBranch(dir string, remoteName string, branch string) error
}
