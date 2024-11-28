package github

import (
	"context"

	"github.com/google/go-github/v62/github"
)

// Commit is a struct that contains the information of a commit
type Commit struct {
	SHA     string       `json:"sha"`
	Message string       `json:"message"`
	Stats   *CommitStats `json:"stats,omitempty"`
	Files   []CommitFile `json:"files,omitempty"`
}

// CommitStats is a struct that contains the statistics of a commit
type CommitStats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Changes   int `json:"changes"`
}

// CommitFile is a struct that contains the information of a commit file
type CommitFile struct {
	Filename string `json:"filename"`
	Patch    string `json:"patch"`
	CommitStats
}

func (client *Client) extractCommitFile(file *github.CommitFile) CommitFile {
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

func (client *Client) extractCommitInformation(ctx context.Context, owner, repository string, originalCommit *github.RepositoryCommit, needCommitDetails bool) (Commit, error) {
	if !needCommitDetails {
		return Commit{
			SHA:     originalCommit.GetSHA(),
			Message: originalCommit.GetCommit().GetMessage(),
		}, nil
	}
	stats := originalCommit.GetStats()
	commitFiles := originalCommit.Files
	if stats == nil || commitFiles == nil {
		commit, _, err := client.Repositories.GetCommit(ctx, owner, repository, originalCommit.GetSHA(), nil)
		if err != nil {
			return Commit{}, addErrMsgToClientError(err)
		}
		// only update stats and files if there is no error
		// otherwise, we will maintain the original commit information
		stats = commit.GetStats()
		commitFiles = commit.Files
	}
	files := make([]CommitFile, len(commitFiles))
	for idx, file := range commitFiles {
		files[idx] = client.extractCommitFile(file)
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
	}, nil
}
