package git

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	survey "gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

type Organisation struct {
	Login string
}

type Repository struct {
	Name             string
	AllowMergeCommit bool
	HTMLURL          string
	CloneURL         string
	SSHURL           string
	Language         string
	Fork             bool
	Stars            int
	URL              string
	Scheme           string
	Host             string
	Organisation     string
	Project          string
}

type PullRequest struct {
	URL            string
	Author         *User
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

type Commit struct {
	SHA       string
	Message   string
	Author    *User
	URL       string
	Branch    string
	Committer *User
}

type Issue struct {
	URL           string
	Owner         string
	Repo          string
	Number        *int
	Key           string
	Title         string
	Body          string
	State         *string
	Labels        []Label
	StatusesURL   *string
	IssueURL      *string
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
	ClosedAt      *time.Time
	IsPullRequest bool
	User          *User
	ClosedBy      *User
	Assignees     []User
}

type User struct {
	URL       string
	Login     string
	Name      string
	Email     string
	AvatarURL string
}

type Release struct {
	Name          string
	TagName       string
	Body          string
	URL           string
	HTMLURL       string
	DownloadCount int
	Assets        *[]ReleaseAsset
}

// ReleaseAsset represents a release stored in Git
type ReleaseAsset struct {
	BrowserDownloadURL string
	Name               string
	ContentType        string
}

type Label struct {
	URL   string
	Name  string
	Color string
}

type RepoStatus struct {
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

type PullRequestArguments struct {
	Title      string
	Body       string
	Head       string
	Base       string
	Repository *Repository
}

type WebhookArguments struct {
	ID     int64
	Owner  string
	Repo   *Repository
	URL    string
	Secret string
}

type FileContent struct {
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

// PullRequestInfo describes a pull request that has been created
type PullRequestInfo struct {
	Provider          Provider
	PullRequest          *PullRequest
	PullRequestArguments *PullRequestArguments
}

// IsClosed returns true if the PullRequest has been closed
func (pr *PullRequest) IsClosed() bool {
	return pr.ClosedAt != nil
}

// NumberString returns the string representation of the Pull Request number or blank if its missing
func (pr *PullRequest) NumberString() string {
	n := pr.Number
	if n == nil {
		return ""
	}
	return "#" + strconv.Itoa(*n)
}

// GetHost returns the Git Provider hostname, e.g github.com
func GetHost(gitProvider Provider) (string, error) {
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

func PickRepositories(provider Provider, owner string, message string, selectAll bool, filter string, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) ([]*Repository, error) {
	answer := []*Repository{}
	repos, err := provider.ListRepositories(owner)
	if err != nil {
		return answer, err
	}

	repoMap := map[string]*Repository{}
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

// IsRepoStatusSuccess returns true if all the statuses are successful
func IsRepoStatusSuccess(statuses ...*RepoStatus) bool {
	for _, status := range statuses {
		if !status.IsSuccess() {
			return false
		}
	}
	return true
}

// IsRepoStatusFailed returns true if any of the statuses have failed
func IsRepoStatusFailed(statuses ...*RepoStatus) bool {
	for _, status := range statuses {
		if status.IsFailed() {
			return true
		}
	}
	return false
}

func (s *RepoStatus) IsSuccess() bool {
	return s.State == "success"
}

func (s *RepoStatus) IsFailed() bool {
	return s.State == "error" || s.State == "failure"
}

// ToLabels converts the list of label names into an array of Labels
func ToLabels(names []string) []Label {
	answer := []Label{}
	for _, n := range names {
		answer = append(answer, Label{Name: n})
	}
	return answer
}
