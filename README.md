# Gateman

A package to authenticate and authorize access.

## Features

- Password hashing and verification using Argon.
- JWT token generation(with custom and registered claims) and parsing.
- HTTP middlewares for auth, rate limiting and middleware composability

## Conventional Commits & Releases

Using [cocogitto](https://github.com/cocogitto/cocogitto) for better experience.

Run these commands:

```sh
# setup git hooks for commit-msg and pre-commit
./scripts/setup-git-hooks.sh

# cog help
cog -h

# generate cog cli completions
cog generate-completions

# create cog.toml file
cog init

# commit changes
cog commit

# bump version & set tag
cog bump -a
```

Based on the [cog.toml](cog.toml) file and/or `cog install-hook -a` command,
commit messages are mandatory conventional.

> Use `cog bump -a` to bump version, set tag and push to remote repository.
