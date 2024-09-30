package github

import (
	"context"

	"github.com/google/go-github/v62/github"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type RepositoriesService interface {
	GetCommit(context.Context, string, string, string, *github.ListOptions) (*github.RepositoryCommit, *github.Response, error)
	CreateHook(context.Context, string, string, *github.Hook) (*github.Hook, *github.Response, error)
}

type Commit struct {
	SHA     string       `json:"sha"`
	Message string       `json:"message"`
	Stats   *CommitStats `json:"stats,omitempty"`
	Files   []CommitFile `json:"files,omitempty"`
}
type CommitStats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Changes   int `json:"changes"`
}
type CommitFile struct {
	Filename string `json:"filename"`
	Patch    string `json:"patch"`
	CommitStats
}

func (githubClient *Client) extractCommitFile(file *github.CommitFile) CommitFile {
	return CommitFile{
		Filename: file.GetFilename(),
		Patch:    file.GetPatch(),
		CommitStats: CommitStats{
			Additions: file.GetAdditions(),
			Deletions: file.GetDeletions(),
			Changes:   file.GetChanges(),
		},
	}
}
func (githubClient *Client) extractCommitInformation(ctx context.Context, owner, repository string, originalCommit *github.RepositoryCommit, needCommitDetails bool) Commit {
	if !needCommitDetails {
		return Commit{
			SHA:     originalCommit.GetSHA(),
			Message: originalCommit.GetCommit().GetMessage(),
		}
	}
	stats := originalCommit.GetStats()
	commitFiles := originalCommit.Files
	if stats == nil || commitFiles == nil {
		commit, err := githubClient.getCommit(ctx, owner, repository, originalCommit.GetSHA())
		if err == nil {
			// only update stats and files if there is no error
			// otherwise, we will maintain the original commit information
			stats = commit.GetStats()
			commitFiles = commit.Files
		}
	}
	files := make([]CommitFile, len(commitFiles))
	for idx, file := range commitFiles {
		files[idx] = githubClient.extractCommitFile(file)
	}
	return Commit{
		SHA:     originalCommit.GetSHA(),
		Message: originalCommit.GetCommit().GetMessage(),
		Stats: &CommitStats{
			Additions: stats.GetAdditions(),
			Deletions: stats.GetDeletions(),
			Changes:   stats.GetTotal(),
		},
		Files: files,
	}
}

func (githubClient *Client) getCommit(ctx context.Context, owner string, repository string, sha string) (*github.RepositoryCommit, error) {
	commit, _, err := githubClient.Repositories.GetCommit(ctx, owner, repository, sha, nil)
	return commit, err
}

type GetCommitInput struct {
	RepoInfo
	SHA string `json:"sha"`
}

type GetCommitResp struct {
	Commit Commit `json:"commit"`
}

func (githubClient *Client) getCommitTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct GetCommitInput
	err := base.ConvertFromStructpb(props, &inputStruct)
	if err != nil {
		return nil, err
	}
	owner, repository, err := parseTargetRepo(inputStruct)
	if err != nil {
		return nil, err
	}
	sha := inputStruct.SHA
	commit, err := githubClient.getCommit(ctx, owner, repository, sha)
	if err != nil {
		return nil, err
	}
	var resp GetCommitResp
	resp.Commit = githubClient.extractCommitInformation(ctx, owner, repository, commit, true)
	out, err := base.ConvertToStructpb(resp)
	if err != nil {
		return nil, err
	}

	return out, nil
}
