# Contributing

Thank you for your interest in contributing to the Terraform Provider for YouTrack!

> [!NOTE]
> This GitHub repository is a read-only mirror. We develop the provider in our internal GitLab and push every change here automatically.

## Issues

If you found a bug or have an idea for a new resource or data source, please open an issue using one of the issue templates. Describe the problem or idea as detailed as possible, including your provider, Terraform and YouTrack version and the configuration to reproduce a bug.

## Pull Requests

Pull requests are welcome, but they cannot be merged on GitHub directly. We review your changes, apply them in our GitLab and they show up here with the next sync. Please keep pull requests small and focused on one change.

## Development

> [!NOTE]
> You need Go 1.25 or later and Terraform installed. The acceptance tests also need Docker.

Build the provider:

```bash
make build
```

Run the tests against the in-memory YouTrack mock:

```bash
make test
```

Run the acceptance tests against a local YouTrack container and remove it afterwards:

```bash
make testacc
make youtrack-down
```

Regenerate the documentation in `docs` after you changed a schema or an example:

```bash
make docs
```

## Commit Messages

Please write commit messages in English and follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification, for example:

```txt
feat: add the youtrack_agile_board resource
fix: keep the order of bundle values after an import
```

## Code of Conduct

Please note that this project is released with a [Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms.
