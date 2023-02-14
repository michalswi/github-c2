package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// base on:
// https://pkg.go.dev/github.com/google/go-github/v50/github#section-documentation
// https://gist.github.com/jaredhoward/f231391529efcd638bb7

const (
	owner = "<repo_owner>"
	repo  = "<repo_name>"
	// where to copy files/directories
	basePath    = "/tmp/"
	accessToken = "<access_token(PAT)>"
)

func main() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	getContents(ctx, client, "")
}

func check(err error) {
	if err != nil {
		log.Println(err)
	}
}

func createDirectory(path string) {
	destination := filepath.Join(basePath, path)
	err := os.Mkdir(destination, 0755)
	check(err)
	fmt.Println("destination path created:", destination)
}

func getContents(ctx context.Context, client *github.Client, path string) {

	_, directoryContent, _, err := client.Repositories.GetContents(ctx, owner, repo, path, nil)
	check(err)

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
	check(err)
	defer rc.Close()

	b, err := ioutil.ReadAll(rc)
	check(err)

	fmt.Println("Writing the file:", localPath)
	f, err := os.Create(localPath)
	check(err)
	defer f.Close()
	n, err := f.Write(b)
	check(err)
	if n != *content.Size {
		fmt.Printf("number of bytes differ, %d vs %d\n", n, *content.Size)
	}
}

func calculateGitSHA1(filePath string) string {
	b, err := ioutil.ReadFile(filePath)
	check(err)
	contentLen := len(b)
	blobSlice := []byte("blob " + strconv.Itoa(contentLen))
	blobSlice = append(blobSlice, '\x00')
	blobSlice = append(blobSlice, b...)
	h := sha1.New()
	h.Write(blobSlice)
	bs := h.Sum(nil)
	return hex.EncodeToString(bs)
}
