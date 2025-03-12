# Contributing to Go Dependency Injection Container

Thank you for considering contributing to our dependency injection container! This document outlines the process for contributing to the project and how to report issues.

## Code of Conduct

By participating in this project, you agree to abide by our code of conduct:

- Be respectful and inclusive
- Be patient and welcoming
- Be thoughtful
- Be collaborative
- When disagreeing, try to understand why

## How to Contribute

### Reporting Bugs

If you find a bug, please create an issue on GitHub with the following information:

1. A clear, descriptive title
2. A detailed description of the issue
3. Steps to reproduce the bug
4. Expected behavior
5. Actual behavior
6. Go version and environment details
7. Code samples if applicable

### Suggesting Enhancements

We welcome suggestions for enhancements! Please create an issue with:

1. A clear, descriptive title
2. A detailed description of the proposed enhancement
3. Any relevant code examples or mockups
4. An explanation of why this enhancement would be useful

### Pull Requests

1. Fork the repository
2. Create a new branch for your feature or bugfix
3. Write tests for your changes
4. Ensure all tests pass
5. Make sure your code follows the existing style
6. Submit a pull request

#### Pull Request Process

1. Update the README.md with details of changes if applicable
2. Update any examples or documentation
3. The PR should work with Go 1.20 and above
4. Include tests that cover your changes
5. Reference any relevant issues in your PR description

## Development Setup

1. Clone the repository
2. Install dependencies: `go mod download`
3. Run tests: `go test ./...`
4. Run benchmarks: `go test -bench=. -benchmem ./...`

## Coding Standards

- Follow standard Go formatting (use `gofmt` or `go fmt`)
- Write descriptive comments for exported functions and types
- Add tests for new functionality
- Maintain backward compatibility when possible
- Use meaningful variable and function names
- Keep functions focused and small

## Testing

- Write unit tests for all new functionality
- Ensure tests are fast and deterministic
- Include both positive and negative test cases
- Test edge cases and error conditions

## Documentation

- Update documentation for any changed functionality
- Add examples for new features
- Keep the README up to date
- Document any breaking changes

## Release Process

Our maintainers will handle the release process, including:

1. Updating version numbers
2. Creating release notes
3. Publishing the new version

## Questions?

If you have any questions about contributing, please open an issue labeled "question".

Thank you for contributing to our project!
