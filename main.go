package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

// Version is set at build time via -ldflags.
var Version = "dev"

const (
	dirMode  = 0o700
	fileMode = 0o600
)

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s is not set", name)
	}
	return v
}

func main() {
	accessToken := requireEnv("GITHUB_PAT")
	owner := requireEnv("REPO_OWNER")
	repo := requireEnv("REPO_NAME")
	basePath := requireEnv("BASE_PATH")

	// optional; empty means the repository's default branch
	var opts *github.RepositoryContentGetOptions
	if branch := os.Getenv("REPO_BRANCH"); branch != "" {
		opts = &github.RepositoryContentGetOptions{Ref: branch}
	}

	log.Printf("gh-c2 version %s", Version)

	// cancelled on SIGINT/SIGTERM so deferred cleanup still runs
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	client := github.NewClient(oauth2.NewClient(ctx, ts))

	if err := getContents(ctx, client, "", owner, repo, basePath, opts); err != nil {
		stop()
		log.Fatalf("Error fetching repository contents: %v", err)
	}
}

// localTarget maps a remote repository path to a path under basePath,
// rejecting anything that would escape it.
func localTarget(basePath, remotePath string) (string, error) {
	if !filepath.IsLocal(filepath.FromSlash(remotePath)) {
		return "", fmt.Errorf("unsafe remote path %q", remotePath)
	}
	return filepath.Join(basePath, filepath.FromSlash(remotePath)), nil
}

func createDirectory(path string, basePath string) error {
	if path == "" {
		return fmt.Errorf("invalid directory path")
	}

	destination, err := localTarget(basePath, path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(destination, dirMode); err != nil {
		return fmt.Errorf("error creating directory %s: %w", destination, err)
	}

	log.Printf("Created directory: %s", destination)
	return nil
}

func getContents(ctx context.Context, client *github.Client, path string, owner string, repo string, basePath string, opts *github.RepositoryContentGetOptions) error {
	_, directoryContent, _, err := client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return fmt.Errorf("error getting contents of %q: %w", path, err)
	}

	var errs []error
	for _, c := range directoryContent {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(errs, err)...)
		}

		log.Println("Processing:", c.GetType(), c.GetPath(), c.GetSize(), c.GetSHA())

		switch c.GetType() {
		case "file":
			local, err := localTarget(basePath, c.GetPath())
			if err != nil {
				errs = append(errs, err)
				continue
			}
			log.Println("Local path:", local)
			if err := handleFile(ctx, client, c, local, owner, repo, opts); err != nil {
				errs = append(errs, fmt.Errorf("file %s: %w", c.GetPath(), err))
			}
		case "dir":
			if err := createDirectory(c.GetPath(), basePath); err != nil {
				errs = append(errs, err)
				continue
			}
			if err := getContents(ctx, client, c.GetPath(), owner, repo, basePath, opts); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func handleFile(ctx context.Context, client *github.Client, content *github.RepositoryContent, localPath string, owner string, repo string, opts *github.RepositoryContentGetOptions) error {
	// Skip the download when the local git blob SHA matches the remote one
	if _, err := os.Stat(localPath); err == nil {
		sha, err := calculateGitSHA1(localPath)
		if err != nil {
			log.Printf("Cannot hash %s, re-downloading: %v", localPath, err)
		} else if content.GetSHA() == sha {
			log.Printf("No need to update file %s, SHA1 is the same", localPath)
			return nil
		}
	}
	return downloadContents(ctx, client, content, localPath, owner, repo, opts)
}

func downloadContents(ctx context.Context, client *github.Client, content *github.RepositoryContent, localPath string, owner string, repo string, opts *github.RepositoryContentGetOptions) (err error) {
	rc, _, err := client.Repositories.DownloadContents(ctx, owner, repo, content.GetPath(), opts)
	if err != nil {
		return err
	}
	defer rc.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), dirMode); err != nil {
		return err
	}

	log.Printf("Writing file: %s", localPath)
	f, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()

	n, err := io.Copy(f, rc)
	if err != nil {
		return err
	}
	if n != int64(content.GetSize()) {
		log.Printf("Warning: written bytes %d do not match expected size %d", n, content.GetSize())
	}

	return nil
}

func calculateGitSHA1(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}

	h := sha1.New()
	h.Write([]byte("blob " + strconv.FormatInt(info.Size(), 10) + "\x00"))
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
