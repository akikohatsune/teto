# Contributing to Teto

First off, thank you for considering contributing to Teto! It's people like you that make open-source projects great.

## Code of Conduct

By participating in this project, you are expected to uphold our [Code of Conduct](CODE_OF_CONDUCT.md). Please report unacceptable behavior to the project maintainers.

## How Can I Contribute?

### Reporting Bugs

If you find a bug, please create an issue and include:

* Your operating system and Go version.
* A clear and descriptive title.
* Steps to reproduce the behavior.
* The expected behavior vs. the actual behavior.
* Any relevant logs or error messages.

### Suggesting Enhancements

If you have an idea for a new feature or improvement, feel free to open an issue! Please provide:

* A clear and descriptive title.
* A detailed explanation of the proposed feature.
* Why this feature would be useful to most users.
* Any potential drawbacks or alternatives considered.

### Submitting Pull Requests

1. **Fork the repository** and create your branch from `main`.
2. **Set up the development environment**:
   * Ensure you have Go 1.20+ installed.
   * Run `go mod tidy` to download dependencies.
3. **Make your changes**:
   * Follow the existing code style.
   * Ensure your code builds without errors (`go build`).
   * Keep your commits focused and provide clear, descriptive commit messages.
4. **Test your changes**: Verify that the bot runs correctly and that no existing functionality is broken.
5. **Open a Pull Request**: Provide a clear title and description referencing any related issues.

## Development Setup

1. Clone your fork: `git clone https://github.com/akikohatsune/teto.git`
2. Enter the directory: `cd github.com/akikohatsune/teto`
3. Download dependencies: `go mod tidy`
4. Copy `.env.example` to `.env` and configure your keys.
5. Run the bot locally: `go run main.go`

Thank you for contributing!

