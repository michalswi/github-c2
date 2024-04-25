## GitHub C2 (command-and-control)

![](https://img.shields.io/github/stars/michalswi/github-c2)
![](https://img.shields.io/github/forks/michalswi/github-c2)
![](https://img.shields.io/github/last-commit/michalswi/github-c2)
![](https://img.shields.io/github/issues/michalswi/github-c2)

Use GitHub as a way to store configuration information.  
Code is **pulling** all file(s) from a GitHub repository to the local file system.

Populate script with:
```
owner       = "<repo_owner>"
repo        = "<repo_name>"
```

Instead of **Classic Tokens** you can use **Fine-grained personal access tokens**. Create a repo (public/private) and generate token (PAT) only for this repo. Set `GITHUB_PAT` env variable.

```
export GITHUB_PAT=github_pat_(...)
```

Script is downloading all the files to your local catalog (default `/tmp`). If you re-run it will compare SHA1 before downloading. If checksum the same, file won't be downloaded.
