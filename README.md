## GitHub C2 (command-and-control)

Use GitHub as a way to store configuration information.

Populate script with:
```
owner       = "<repo_owner>"
repo        = "<repo_name>"
accessToken = "<access_token(PAT)>"
```

Instead of **Classic Tokens** you can use **Fine-grained personal access tokens**. Create private repo and generate token (PAT) only for this repo.

Script is downloading all the files to your local catalog (default `/tmp`). If you re-run it will compare SHA1 before downloading. If checksum the same, file won't be downloaded.
