package picker

import (
	"github.com/jenkins-x/jx/pkg/auth"
	"github.com/wbrefvem/go-gits/pkg/bitbucket"
	"github.com/wbrefvem/go-gits/pkg/gitea"
	"github.com/wbrefvem/go-gits/pkg/github"
	"github.com/wbrefvem/go-gits/pkg/gitlab"
	"github.com/wbrefvem/go-gits/pkg/gitter"
)

func GetProviderConstructor(provider string) func(server *auth.AuthServer, user *auth.UserAuth, git gitter.Gitter) (git.GitProvider, error) {
	if server.Kind == KindBitBucketCloud {
		return bitbucket.NewBitbucketCloudProvider
	} else if server.Kind == KindBitBucketServer {
		return bitbucket.NewBitbucketServerProvider
	} else if server.Kind == KindGitea {
		return gitea.NewGiteaProvider
	} else if server.Kind == KindGitlab {
		return gitlab.NewGitlabProvider
	} else if server.Kind == KindGitFake {
		return NewFakeGitProvider
	} else {
		return github.NewGitHubProvider
	}
}
