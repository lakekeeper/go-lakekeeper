# Contributing to go-lakekeeper

Thank you for your interest in contributing to **go-lakekeeper**! 🎉  
We welcome issues, bug fixes, new features, and documentation improvements.

Please take a moment to review this guide before submitting a pull request.

## Table of Contents

- [Contributing to go-lakekeeper](#contributing-to-go-lakekeeper)
  - [Table of Contents](#table-of-contents)
  - [Code of Conduct](#code-of-conduct)
  - [Getting Started](#getting-started)
  - [How to Contribute](#how-to-contribute)
    - [Reporting Bugs](#reporting-bugs)
    - [Suggesting Features](#suggesting-features)
    - [Submitting Changes](#submitting-changes)
  - [Coding Guidelines](#coding-guidelines)
  - [Commits and branches](#commits-and-branches)
  - [Testing](#testing)
  - [Pull Request Process](#pull-request-process)
  - [License](#license)

---

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md) to help create a welcoming and respectful community.

## Getting Started

1. Fork the repository and clone it locally.
2. Install [Go 1.26+](https://golang.org/dl/).
3. Run `go mod tidy` to install dependencies.
4. Run `make build` or use your preferred method to build the CLI/client.

## How to Contribute

### Reporting Bugs

- Use [GitHub Issues](https://github.com/lakekeeper/go-lakekeeper/issues).
- Include a clear title and description.
- Include steps to reproduce, expected behavior, and actual behavior.

### Suggesting Features

- Open an issue
- Provide context and possible use cases.

### Submitting Changes

- Fork the repo and create a new branch: `git checkout -b feat/my-feature`
- Make your changes
- Format your code: `make fmt`
- Run tests: `make test`
- Commit and push your branch
- Open a pull request and follow the PR template

## Coding Guidelines

- Use `gofmt`, `golangci-lint`, and `go vet` before submitting.
- Keep your changes focused: small, atomic PRs are easier to review.
- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines.

## Commits and branches

- Commits and pull request titles follow [Conventional Commits](https://www.conventionalcommits.org), e.g. `fix: handle empty role list` or `feat(storage): support GCS`. Start the description with a lowercase letter, use the imperative mood, and leave off the trailing period; this is convention, not checked by CI. CI checks the PR title's format and accepts the types `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, and `test`; [release-please](https://github.com/googleapis/release-please) derives versions and `CHANGELOG.md` from it.
- Mark a breaking change with `!` (e.g. `feat!: rename client option`) and describe the migration in the PR.
- Name branches `type/short-description` using the commit type, e.g. `fix/role-list-pagination`.
- `CHANGELOG.md` and `.release-please-manifest.json` are maintained by release-please; leave them untouched.

## Regenerating the API client

The Management API client (`pkg/apis/management/v1/`) is generated from
the upstream OpenAPI spec — **do not hand-edit any file in that
directory.** The full regeneration workflow, including the OpenAPI
preprocessor and the generated/manual file boundary, is documented in
[docs/GENERATION.md](docs/GENERATION.md). The short version:

```sh
make generate
```

## Testing

Run all tests before opening a PR:

```sh
# go vet + golangci-lint
make validate

# Unit tests
make test

# Integration tests
make test-integration

# lkctl end-to-end tests against the compose stack
make test-e2e-compose
```

`make test-e2e-kind` runs the `lkctl` end-to-end tests against a kind cluster; it needs `kind`, `kubectl` and `helm` installed.

If adding a feature or fixing a bug, include relevant unit and/or integration tests.

## Pull Request Process

1. Ensure all CI checks pass (unit tests, linters, etc.)
2. Make sure your branch is up to date with `main`
3. Link related issues in the PR description
4. A maintainer will review your PR and may request changes
5. Once approved, your PR will be merged into `main`

## License

By contributing, you agree that your contributions will be licensed under the [Apache 2.0 License](LICENSE).

---

Thanks again for your support and contributions!
Let's build something great together.