package util

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var githubClient *github.Client

// Download a file from the given URL
func DownloadFile(filepath string, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := GetClientWithTimeout(time.Duration(time.Hour * 2)).Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("download of %s failed with return code %d", url, resp.StatusCode)
		return err
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// make it executable
	os.Chmod(filepath, 0755)
	if err != nil {
		return err
	}
	return nil
}

func GetLatestVersionFromGitHub(githubOwner, githubRepo string) (semver.Version, error) {
	text, err := GetLatestVersionStringFromGitHub(githubOwner, githubRepo)
	if err != nil {
		return semver.Version{}, err
	}
	if text == "" {
		return semver.Version{}, fmt.Errorf("No version found")
	}
	return semver.Make(text)
}

func GetLatestVersionStringFromGitHub(githubOwner, githubRepo string) (string, error) {
	latestVersionString, err := GetLatestReleaseFromGitHub(githubOwner, githubRepo)
	if err != nil {
		return "", err
	}
	if latestVersionString != "" {
		return strings.TrimPrefix(latestVersionString, "v"), nil
	}
	return "", fmt.Errorf("Unable to find the latest version for github.com/%s/%s", githubOwner, githubRepo)
}

// GetLatestReleaseFromGitHub gets the latest Release from a specific github repo
func GetLatestReleaseFromGitHub(githubOwner, githubRepo string) (string, error) {
	client, release, resp, err := preamble()
	release, resp, err = client.Repositories.GetLatestRelease(context.Background(), githubOwner, githubRepo)
	if err != nil {
		return "", fmt.Errorf("Unable to get latest version for github.com/%s/%s %v", githubOwner, githubRepo, err)
	}
	defer resp.Body.Close()
	latestVersionString := release.TagName
	if latestVersionString != nil {
		return *latestVersionString, nil
	}
	return "", fmt.Errorf("Unable to find the latest version for github.com/%s/%s", githubOwner, githubRepo)
}

// GetLatestFullTagFromGithub gets the latest 'full' tag from a specific github repo. This (at present) ignores releases
// with a hyphen in it, usually used with -SNAPSHOT, or -RC1 or -beta
func GetLatestFullTagFromGithub(githubOwner, githubRepo string) (string, error) {
	tags, err := GetTagsFromGithub(githubOwner, githubRepo)
	if err == nil {
		// Iterate over the tags to find the first that doesn't contain any hyphens in it (so is just x.y.z)
		for _, tag := range tags {
			name := *tag.Name
			if !strings.ContainsRune(name, '-') {
				return name, nil
			}
		}
		return "", errors.Errorf("No Full releases found for %s/%s", githubOwner, githubRepo)
	}
	return "", err
}

// GetLatestTagFromGithub gets the latest (in github order) tag from a specific github repo
func GetLatestTagFromGithub(githubOwner, githubRepo string) (string, error) {
	tags, err := GetTagsFromGithub(githubOwner, githubRepo)
	if err == nil {
		return *tags[0].Name, nil
	}
	return "", err
}

// GetTagsFromGithub gets the list of tags on a specific github repo
func GetTagsFromGithub(githubOwner, githubRepo string) ([]*github.RepositoryTag, error) {
	client, _, resp, err := preamble()

	tags, resp, err := client.Repositories.ListTags(context.Background(), githubOwner, githubRepo, nil)
	defer resp.Body.Close()
	if err != nil {
		return []*github.RepositoryTag{}, fmt.Errorf("Unable to get tags for github.com/%s/%s %v", githubOwner, githubRepo, err)
	}

	return tags, nil
}

func preamble() (*github.Client, *github.RepositoryRelease, *github.Response, error) {
	if githubClient == nil {
		token := os.Getenv("GH_TOKEN")
		var tc *http.Client
		if len(token) > 0 {
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: token},
			)
			tc = oauth2.NewClient(oauth2.NoContext, ts)
		}
		githubClient = github.NewClient(tc)
	}
	client := githubClient
	var (
		release *github.RepositoryRelease
		resp    *github.Response
		err     error
	)
	return client, release, resp, err
}

// untargz a tarball to a target, from
// http://blog.ralch.com/tutorial/golang-working-with-tar-and-gzipf
func UnTargz(tarball, target string, onlyFiles []string) error {
	zreader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer zreader.Close()

	reader, err := gzip.NewReader(zreader)
	defer reader.Close()
	if err != nil {
		panic(err)
	}

	tarReader := tar.NewReader(reader)

	for {
		inkey := false
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		for _, value := range onlyFiles {
			if value == "*" || value == path.Base(header.Name) {
				inkey = true
				break
			}
		}

		if !inkey && len(onlyFiles) > 0 {
			continue
		}

		path := filepath.Join(target, path.Base(header.Name))
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}

// untargz a tarball to a target including any folders inside the tarball
// http://blog.ralch.com/tutorial/golang-working-with-tar-and-gzipf
func UnTargzAll(tarball, target string) error {
	zreader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer zreader.Close()

	reader, err := gzip.NewReader(zreader)
	defer reader.Close()
	if err != nil {
		panic(err)
	}

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}
