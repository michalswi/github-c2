package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

func main() {
	accessToken := os.Getenv("GITHUB_PAT")
	if accessToken == "" {
		log.Fatal("GITHUB_PAT is not set")
	}

	owner := os.Getenv("REPO_OWNER")
	if accessToken == "" {
		log.Fatal("REPO_OWNER is not set")
	}

	repo := os.Getenv("REPO_NAME")
	if accessToken == "" {
		log.Fatal("REPO_NAME is not set")
	}

	basePath := os.Getenv("BASE_PATH")
	if accessToken == "" {
		log.Fatal("BASE_PATH is not set")
	}

	// graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	go func() {
		<-signalChan
		fmt.Println("Interrupt signal received, cleaning up...")
		cancel()
		os.Exit(0)
	}()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)

	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	err := getContents(ctx, client, "", owner, repo, basePath)
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

func createDirectory(path string, basePath string) error {
	if path == "" {
		return fmt.Errorf("invalid directory path")
	}

	destination := filepath.Join(basePath, path)

	err := os.Mkdir(destination, 0755)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("error creating directory %s: %w", destination, err)
	}

	log.Printf("Created directory: %s", destination)
	return nil
}

func getContents(ctx context.Context, client *github.Client, path string, owner string, repo string, basePath string) error {
	_, directoryContent, _, err := client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if check(err) {
		return err
	}

	for _, c := range directoryContent {
		log.Println("Processing:", *c.Type, *c.Path, *c.Size, *c.SHA)

		local := filepath.Join(basePath, *c.Path)
		log.Println("Local path:", local)

		switch *c.Type {
		case "file":
			err := handleFile(ctx, client, c, local, owner, repo, basePath)
			if err != nil {
				log.Printf("Error handling file: %v", err)
			}
		case "dir":
			err := createDirectory(*c.Path, basePath)
			if err != nil {
				log.Printf("Error creating directory: %v", err)
			}
			getContents(ctx, client, *c.Path, owner, repo, basePath)
		}
	}
	return nil
}

func handleFile(ctx context.Context, client *github.Client, content *github.RepositoryContent, localPath string, owner string, repo string, basePath string) error {
	// Check if file exists and compare SHA1
	_, err := os.Stat(localPath)
	if err == nil {
		sha := calculateGitSHA1(localPath)
		if *content.SHA == sha {
			log.Printf("No need to update file %s, SHA1 is the same", localPath)
			return nil
		}
	}
	return downloadContents(ctx, client, content, localPath, owner, repo)
}

func downloadContents(ctx context.Context, client *github.Client, content *github.RepositoryContent, localPath string, owner string, repo string) error {
	rc, _, err := client.Repositories.DownloadContents(ctx, owner, repo, *content.Path, nil)
	if check(err) {
		return err
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
	if check(err) {
		return err
	}

	log.Printf("Writing file: %s", localPath)
	f, err := os.Create(localPath)
	if check(err) {
		return err
	}
	defer f.Close()

	n, err := f.Write(b)
	if check(err) {
		return err
	}
	if n != *content.Size {
		log.Printf("Warning: written bytes %d do not match expected size %d", n, *content.Size)
	}

	return nil
}

func calculateGitSHA1(filePath string) string {
	b, err := os.ReadFile(filePath)
	if check(err) {
		return ""
	}

	contentLen := len(b)
	blobSlice := []byte("blob " + strconv.Itoa(contentLen))
	blobSlice = append(blobSlice, '\x00')
	blobSlice = append(blobSlice, b...)

	h := sha1.New()
	h.Write(blobSlice)
	return hex.EncodeToString(h.Sum(nil))
}
