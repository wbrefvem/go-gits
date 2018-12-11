package gits

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/auth"
	gitcfg "gopkg.in/src-d/go-git.v4/config"
)

// GitRemote a remote url
type GitRemote struct {
	Name string
	URL  string
}

// GitTag git tag
type GitTag struct {
	Name    string
	Message string
}

// GitFake provides a fake Gitter
type GitFake struct {
	Remotes        []GitRemote
	Branches       []string
	BranchesRemote []string
	CurrentBranch  string
	AccessTokenURL string
	RepoInfo       GitRepository
	Fork           bool
	GitVersion     string
	GitUser        GitUser
	Commits        []GitCommit
	Changes        bool
	GitTags        []GitTag
	Revision       string
	serverURL      string
}

// NewGitFake creates a new fake Gitter
func NewGitFake() Gitter {
	return &GitFake{}
}

// FindGitConfigDir finds the git config dir
func (g *GitFake) FindGitConfigDir(dir string) (string, string, error) {
	return dir, dir, nil
}

// PrintCreateRepositoryGenerateAccessToken prints the generate access token URL
func (g *GitFake) PrintCreateRepositoryGenerateAccessToken(server *auth.AuthServer, username string, o io.Writer) {
	fmt.Fprintf(o, "Access token URL: %s\n\n", g.AccessTokenURL)
}

// Status check the status
func (g *GitFake) Status(dir string) error {
	return nil
}

// Server returns the server URL
func (g *GitFake) Server(dir string) (string, error) {
	return g.RepoInfo.HostURL(), nil
}

// Info returns the git repo info
func (g *GitFake) Info(dir string) (*GitRepository, error) {
	return &g.RepoInfo, nil
}

// IsFork returns trie if this repo is a fork
func (g *GitFake) IsFork(dir string) (bool, error) {
	return g.Fork, nil
}

// Version returns the git version
func (g *GitFake) Version() (string, error) {
	return g.GitVersion, nil
}

// RepoName returns the repo name
func (g *GitFake) RepoName(org string, repoName string) string {
	if org != "" {
		return org + "/" + repoName
	}
	return repoName
}

// Username returns the current user name
func (g *GitFake) Username(dir string) (string, error) {
	return g.GitUser.Name, nil
}

// SetUsername sets the username
func (g *GitFake) SetUsername(dir string, username string) error {
	g.GitUser.Name = username
	return nil
}

// Email returns the current user git email address
func (g *GitFake) Email(dir string) (string, error) {
	return g.GitUser.Email, nil
}

// SetEmail sets the git email address
func (g *GitFake) SetEmail(dir string, email string) error {
	g.GitUser.Email = email
	return nil
}

// GetAuthorEmailForCommit returns the author email for a commit
func (g *GitFake) GetAuthorEmailForCommit(dir string, sha string) (string, error) {
	for _, commit := range g.Commits {
		if commit.SHA == sha {
			return commit.Author.Email, nil
		}
	}
	return "", errors.New("No commit found with given SHA")
}

// Init initialises git in a dir
func (g *GitFake) Init(dir string) error {
	return nil
}

// Clone clones the repo to the given dir
func (g *GitFake) Clone(url string, directory string) error {
	return nil
}

// ShallowCloneBranch shallow clone of a branch
func (g *GitFake) ShallowCloneBranch(url string, branch string, directory string) error {
	return nil
}

// Push performs a git push
func (g *GitFake) Push(dir string) error {
	return nil
}

// PushMaster pushes to master
func (g *GitFake) PushMaster(dir string) error {
	return nil
}

// PushTag pushes a tag
func (g *GitFake) PushTag(dir string, tag string) error {
	return nil
}

// CreatePushURL creates a Push URL
func (g *GitFake) CreatePushURL(cloneURL string, userAuth *auth.UserAuth) (string, error) {
	u, err := url.Parse(cloneURL)
	if err != nil {
		return cloneURL, nil
	}
	if userAuth.Username != "" || userAuth.ApiToken != "" {
		u.User = url.UserPassword(userAuth.Username, userAuth.ApiToken)
		return u.String(), nil
	}
	return cloneURL, nil
}

// ForcePushBranch force push a branch
func (g *GitFake) ForcePushBranch(dir string, localBranch string, remoteBranch string) error {
	return nil
}

// CloneOrPull performs a clone or pull
func (g *GitFake) CloneOrPull(url string, directory string) error {
	return nil
}

// Pull git pulls
func (g *GitFake) Pull(dir string) error {
	return nil
}

// PullRemoteBranches pull remote branches
func (g *GitFake) PullRemoteBranches(dir string) error {
	return nil
}

// PullUpstream pulls upstream
func (g *GitFake) PullUpstream(dir string) error {
	return nil
}

// AddRemote adds a remote
func (g *GitFake) AddRemote(dir string, name string, url string) error {
	r := GitRemote{
		Name: name,
		URL:  url,
	}
	g.Remotes = append(g.Remotes, r)
	return nil
}

func (g *GitFake) findRemote(name string) (*GitRemote, error) {
	for _, remote := range g.Remotes {
		if remote.Name == name {
			return &remote, nil
		}
	}
	return nil, fmt.Errorf("no remote found with name '%s'", name)
}

// SetRemoteURL sets a remote URL
func (g *GitFake) SetRemoteURL(dir string, name string, gitURL string) error {
	remote, err := g.findRemote(name)
	if err != nil {
		r := GitRemote{
			Name: name,
			URL:  gitURL,
		}
		g.Remotes = append(g.Remotes, r)
		return nil
	}
	remote.URL = gitURL
	return nil
}

// UpdateRemote updates a remote
func (g *GitFake) UpdateRemote(dir string, url string) error {
	return g.SetRemoteURL(dir, "origin", url)
}

// DeleteRemoteBranch deletes a remote branch
func (g *GitFake) DeleteRemoteBranch(dir string, remoteName string, branch string) error {
	return nil
}

// DiscoverRemoteGitURL discover the remote git URL
func (g *GitFake) DiscoverRemoteGitURL(gitConf string) (string, error) {
	origin, err := g.findRemote("origin")
	if err != nil {
		upstream, err := g.findRemote("upstream")
		if err != nil {
			return "", err
		}
		return upstream.URL, nil
	}
	return origin.URL, nil
}

// DiscoverUpstreamGitURL discover the upstream git URL
func (g *GitFake) DiscoverUpstreamGitURL(gitConf string) (string, error) {
	upstream, err := g.findRemote("upstream")
	if err != nil {
		origin, err := g.findRemote("origin")
		if err != nil {
			return "", err
		}
		return origin.URL, nil
	}
	return upstream.URL, nil
}

// RemoteBranches list the remote branches
func (g *GitFake) RemoteBranches(dir string) ([]string, error) {
	return g.BranchesRemote, nil
}

// RemoteBranchNames list the remote branch names
func (g *GitFake) RemoteBranchNames(dir string, prefix string) ([]string, error) {
	remoteBranches := []string{}
	for _, remoteBranch := range g.BranchesRemote {
		remoteBranch = strings.TrimSpace(strings.TrimPrefix(remoteBranch, "* "))
		if prefix != "" {
			remoteBranch = strings.TrimPrefix(remoteBranch, prefix)
		}
		remoteBranches = append(remoteBranches, remoteBranch)
	}
	return remoteBranches, nil
}

// GetRemoteUrl get the remote URL
func (g *GitFake) GetRemoteUrl(config *gitcfg.Config, name string) string {
	if len(g.Remotes) == 0 {
		return ""
	}
	return g.Remotes[0].URL
}

// Branch returns the current branch
func (g *GitFake) Branch(dir string) (string, error) {
	return g.CurrentBranch, nil
}

// CreateBranch creates a branch
func (g *GitFake) CreateBranch(dir string, branch string) error {
	g.Branches = append(g.Branches, branch)
	return nil
}

// CheckoutRemoteBranch checkout remote branch
func (g *GitFake) CheckoutRemoteBranch(dir string, branch string) error {
	g.CurrentBranch = branch
	g.Branches = append(g.Branches, branch)
	return nil
}

// Checkout checkout the branch
func (g *GitFake) Checkout(dir string, branch string) error {
	g.CurrentBranch = branch
	return nil
}

// CheckoutOrphan checkout the orphan
func (g *GitFake) CheckoutOrphan(dir string, branch string) error {
	g.CurrentBranch = branch
	return nil
}

// ConvertToValidBranchName converts the name to a valid branch name
func (g *GitFake) ConvertToValidBranchName(name string) string {
	name = strings.TrimSuffix(name, "/")
	name = strings.TrimSuffix(name, ".lock")
	var buffer bytes.Buffer

	last := ' '
	for _, ch := range name {
		if ch <= 32 {
			ch = replaceInvalidBranchChars
		}
		switch ch {
		case '~':
			ch = replaceInvalidBranchChars
		case '^':
			ch = replaceInvalidBranchChars
		case ':':
			ch = replaceInvalidBranchChars
		case ' ':
			ch = replaceInvalidBranchChars
		case '\n':
			ch = replaceInvalidBranchChars
		case '\r':
			ch = replaceInvalidBranchChars
		case '\t':
			ch = replaceInvalidBranchChars
		}
		if ch != replaceInvalidBranchChars || last != replaceInvalidBranchChars {
			buffer.WriteString(string(ch))
		}
		last = ch
	}
	return buffer.String()
}

// FetchBranch fetch branch
func (g *GitFake) FetchBranch(dir string, repo string, refspec string) error {
	return nil
}

// Stash git stash
func (g *GitFake) Stash(dir string) error {
	return nil
}

// Remove a file from git
func (g *GitFake) Remove(dir string, fileName string) error {
	return nil
}

// RemoveForce remove force
func (g *GitFake) RemoveForce(dir string, fileName string) error {
	return nil
}

// CleanForce clean force
func (g *GitFake) CleanForce(dir string, fileName string) error {
	return nil
}

// Add add files to git
func (g *GitFake) Add(dir string, args ...string) error {
	return nil
}

// CommitIfChanges git commit if there are changes
func (g *GitFake) CommitIfChanges(dir string, message string) error {
	commit := GitCommit{
		SHA:       "",
		Message:   message,
		Author:    &g.GitUser,
		URL:       g.RepoInfo.URL,
		Branch:    g.CurrentBranch,
		Committer: &g.GitUser,
	}
	g.Commits = append(g.Commits, commit)
	return nil
}

// CommitDir commit a dir
func (g *GitFake) CommitDir(dir string, message string) error {
	return g.CommitIfChanges(dir, message)
}

// AddCommit add a commit
func (g *GitFake) AddCommit(dir string, msg string) error {
	return g.CommitIfChanges(dir, msg)
}

// HasChanges returns true if has changes in git
func (g *GitFake) HasChanges(dir string) (bool, error) {
	return g.Changes, nil
}

// GetPreviousGitTagSHA returns the previous git tag SHA
func (g *GitFake) GetPreviousGitTagSHA(dir string) (string, error) {
	len := len(g.Commits)
	if len < 2 {
		return "", errors.New("no previous commit found")
	}
	return g.Commits[len-2].SHA, nil
}

// GetCurrentGitTagSHA returns the current git tag sha
func (g *GitFake) GetCurrentGitTagSHA(dir string) (string, error) {
	len := len(g.Commits)
	if len < 1 {
		return "", errors.New("no current commit found")
	}
	return g.Commits[len-1].SHA, nil
}

// GetLatestCommitMessage returns the last commit message
func (g *GitFake) GetLatestCommitMessage(dir string) (string, error) {
	len := len(g.Commits)
	if len < 1 {
		return "", errors.New("no current commit found")
	}
	return g.Commits[len-1].Message, nil
}

// FetchTags fetches tags
func (g *GitFake) FetchTags(dir string) error {
	return nil
}

// Tags lists the tags
func (g *GitFake) Tags(dir string) ([]string, error) {
	tags := []string{}
	for _, tag := range g.GitTags {
		tags = append(tags, tag.Name)
	}
	return tags, nil
}

// CreateTag creates a tag
func (g *GitFake) CreateTag(dir string, tag string, msg string) error {
	t := GitTag{
		Name:    tag,
		Message: msg,
	}
	g.GitTags = append(g.GitTags, t)
	return nil
}

// GetRevisionBeforeDate get the revision before the date
func (g *GitFake) GetRevisionBeforeDate(dir string, t time.Time) (string, error) {
	return g.Revision, nil
}

// GetRevisionBeforeDateText get the revision before the date text
func (g *GitFake) GetRevisionBeforeDateText(dir string, dateText string) (string, error) {
	return g.Revision, nil
}

// Diff performs a git diff
func (g *GitFake) Diff(dir string) (string, error) {
	return "", nil
}

func (g *GitFake) notFound() error {
	return fmt.Errorf("Not found")
}
