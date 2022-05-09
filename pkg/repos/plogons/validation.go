package plogons

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v44/github"
)

func ValidatePullRequest(pr *github.PullRequest) (*PlogonMetaValidationResult, error) {
	res := &PlogonMetaValidationResult{}

	files, _, err := downloadGitDiff(pr)
	if err != nil {
		return nil, err
	}

	uncompressedMeta, err := downloadMeta(pr, files)
	if err != nil {
		return nil, err
	}

	compressedMeta, err := downloadZippedMeta(pr, files)
	if err != nil {
		return nil, err
	}

	// Check PR title if this PR is targetting testing
	for _, f := range files {
		if strings.Contains(f.NewName, "testing") {
			res.Testing = true
			break
		}
	}

	tags := getTags(pr.GetTitle())
	for _, t := range tags {
		if strings.ToLower(t) == "testing" {
			res.TestingHasTaggedTitle = true
			break
		}
	}

	// Test basic fields
	if uncompressedMeta.Name != "" {
		res.NameSet = true
	}

	if uncompressedMeta.InternalName != "" {
		res.InternalNameSet = true
	}

	if uncompressedMeta.Description != "" {
		res.DescriptionSet = true
	}

	if uncompressedMeta.AssemblyVersion != "" {
		res.AssemblyVersionSet = true
	}

	if uncompressedMeta.RepoURL != "" {
		res.RepoURLSet = true
	}

	if uncompressedMeta.Punchline != "" {
		res.PunchlineSet = true
	}

	// Check that the two metadata files are equivalent
	if cmp.Equal(*uncompressedMeta, *compressedMeta) {
		res.MatchesZipped = true
	}

	return res, nil
}

func downloadMeta(pr *github.PullRequest, diffFiles []*gitdiff.File) (*PlogonMeta, error) {
	meta := &PlogonMeta{}

	metaFileInfo := findMetaFile(diffFiles)
	if metaFileInfo == nil {
		return nil, fmt.Errorf("could not find metadata file in pull request")
	}

	metaFileURL, err := getHeadBranchFileURL(metaFileInfo, pr)
	if err != nil {
		return nil, err
	}

	metaFile, err := http.Get(metaFileURL)
	if err != nil {
		return nil, err
	}
	defer metaFile.Body.Close()

	metaFileBuf, err := ioutil.ReadAll(metaFile.Body)
	if err != nil {
		return nil, err
	}

	metaFileData := strings.TrimLeftFunc(string(metaFileBuf), func(r rune) bool {
		return r != '{'
	})
	metaFileData = strings.TrimRightFunc(metaFileData, func(r rune) bool {
		return r != '}'
	})

	err = json.Unmarshal([]byte(metaFileData), meta)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func downloadZippedMeta(pr *github.PullRequest, diffFiles []*gitdiff.File) (*PlogonMeta, error) {
	meta := &PlogonMeta{}

	zipFileInfo := findZipFile(diffFiles)
	if zipFileInfo == nil {
		return nil, fmt.Errorf("could not find zip file in pull request")
	}

	zipFileURL, err := getHeadBranchFileURL(zipFileInfo, pr)
	if err != nil {
		return nil, err
	}

	zipFile, err := http.Get(zipFileURL)
	if err != nil {
		return nil, err
	}
	defer zipFile.Body.Close()

	zipFileBuf, err := ioutil.ReadAll(zipFile.Body)
	if err != nil {
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipFileBuf), zipFile.ContentLength)
	if err != nil {
		return nil, err
	}

	var metaFile *zip.File
	for _, zipFile := range zipReader.File {
		if strings.HasSuffix(zipFile.Name, ".json") {
			metaFile = zipFile
		}
	}

	if metaFile == nil {
		return nil, fmt.Errorf("could not find metadata file in zip")
	}

	metaFileContents, err := metaFile.Open()
	if err != nil {
		return nil, err
	}
	defer metaFileContents.Close()

	metaFileBuf, err := ioutil.ReadAll(metaFileContents)
	if err != nil {
		return nil, err
	}

	metaFileData := strings.TrimLeftFunc(string(metaFileBuf), func(r rune) bool {
		return r != '{'
	})
	metaFileData = strings.TrimRightFunc(metaFileData, func(r rune) bool {
		return r != '}'
	})

	err = json.Unmarshal([]byte(metaFileData), meta)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func getHeadBranchFileURL(file *gitdiff.File, pr *github.PullRequest) (string, error) {
	if pr.Head == nil {
		return "", fmt.Errorf("pull request has nil head branch")
	}

	if pr.Head.Repo == nil {
		return "", fmt.Errorf("pull request branch has nil repo")
	}

	fileURL, err := url.Parse("https://raw.githubusercontent.com")
	if err != nil {
		return "", err
	}

	fileURL.Path = path.Join(fileURL.Path, pr.Head.Repo.GetFullName(),
		pr.Head.GetRef(), strings.TrimLeft(file.NewName, "b/"))

	return fileURL.String(), nil
}

func getTags(title string) []string {
	if !strings.HasPrefix(title, "[") {
		return nil
	}

	tags := make([]string, 0)
	openPos := 0
	closeFound := false
	for i, c := range title {
		if c == ']' {
			if !closeFound {
				tags = append(tags, title[openPos+1:i])
			}

			closeFound = true
		} else if c == '[' {
			if closeFound {
				closeFound = false
			}

			openPos = i
		}
	}

	return tags
}

func findZipFile(diffFiles []*gitdiff.File) *gitdiff.File {
	var fileInfo *gitdiff.File
	for _, f := range diffFiles {
		if strings.HasSuffix(f.NewName, ".zip") {
			fileInfo = f
			break
		}
	}

	return fileInfo
}

func findMetaFile(diffFiles []*gitdiff.File) *gitdiff.File {
	var fileInfo *gitdiff.File
	for _, f := range diffFiles {
		if strings.HasSuffix(f.NewName, ".json") {
			fileInfo = f
			break
		}
	}

	return fileInfo
}

func downloadGitDiff(pr *github.PullRequest) ([]*gitdiff.File, string, error) {
	diffURL := pr.GetDiffURL()
	diff, err := http.Get(diffURL)
	if err != nil {
		return nil, "", err
	}
	defer diff.Body.Close()

	files, preamble, err := gitdiff.Parse(diff.Body)
	if err != nil {
		return nil, "", err
	}

	return files, preamble, nil
}
