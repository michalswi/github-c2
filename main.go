package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

// based on:
// https://github.com/google/go-github
// https://pkg.go.dev/github.com/google/go-github/v66/github

// todo - move to env vars
// const (
// 	owner    = "<repo_owner>"
// 	repo     = "<repo_name>"
// 	basePath = "/tmp" // where to download files
// )

func main() {
	accessToken := os.Getenv("GITHUB_PAT")
	if accessToken == "" {
		log.Fatal("GITHUB_PAT is not set")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	err := getContents(ctx, client, "")
	if err != nil {
		log.Fatalf("Error fetching repository contents: %v", err)
	}
}

func check(err error) bool {
	if err != nil {
		log.Println("Error:", err)
		return true
	}
	return false
}

func createDirectory(path string) {
	destination := filepath.Join(basePath, path)
	err := os.Mkdir(destination, 0755)
	if check(err) != nil {
		return
	}
	fmt.Println("destination path created:", destination)
}

func getContents(ctx context.Context, client *github.Client, path string) error {

	_, directoryContent, _, err := client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if check(err) {
		return err
	}

	for _, c := range directoryContent {
		fmt.Println("file/dir details:", *c.Type, *c.Path, *c.Size, *c.SHA)

		local := filepath.Join(basePath, *c.Path)
		fmt.Println("file/dir path:", local)

		switch *c.Type {
		case "file":
			_, err := os.Stat(local)
			if err == nil {
				sha := calculateGitSHA1(local)
				// fmt.Println(*c.SHA)
				// fmt.Println(sha)
				if *c.SHA == sha {
					fmt.Println("No need to update this file, the SHA1 is the same")
					continue
				}
			}
			downloadContents(ctx, client, c, local)
		case "dir":
			createDirectory(*c.Path)
			getContents(ctx, client, *c.Path)
		}
	}
}

func downloadContents(ctx context.Context, client *github.Client, content *github.RepositoryContent, localPath string) {
	if content.Content != nil {
		fmt.Println("content:", *content.Content)
	}

	rc, _, err := client.Repositories.DownloadContents(ctx, owner, repo, *content.Path, nil)
	if check(err) != nil {
		return
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
	if check(err) != nil {
		return
	}

	fmt.Println("Writing the file:", localPath)
	f, err := os.Create(localPath)
	if check(err) != nil {
		return
	}
	defer f.Close()
	n, err := f.Write(b)
	if check(err) != nil {
		return
	}
	if n != *content.Size {
		fmt.Printf("number of bytes differ, %d vs %d\n", n, *content.Size)
	}
}

func calculateGitSHA1(filePath string) string {
	b, err := os.ReadFile(filePath)
	if check(err) != nil {
		return ""
	}
	contentLen := len(b)
	blobSlice := []byte("blob " + strconv.Itoa(contentLen))
	blobSlice = append(blobSlice, '\x00')
	blobSlice = append(blobSlice, b...)
	h := sha1.New()
	h.Write(blobSlice)
	bs := h.Sum(nil)
	return hex.EncodeToString(bs)
}
