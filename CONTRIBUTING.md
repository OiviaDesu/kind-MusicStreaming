# Contributing to OiviaKind (MusicService Operator)

Thank you for your interest in contributing! 🎵

## Getting Started

### 1. Fork and Clone

```sh
git clone https://github.com/<your-username>/OiviaKind.git
cd OiviaKind
```

### 2. Environment Setup

```sh
# Copy environment template
cp .env.example .env

# Edit with your configuration
nano .env
```

⚠️ **NEVER commit `.env` file** - it contains sensitive data!

### 3. Install Dependencies

```sh
# Install Go dependencies
go mod download

# Install development tools
make install-tools
```

### 4. Run Tests

```sh
# Run unit tests
make test

# Run e2e tests (requires Kind cluster)
make test-e2e

# Check test coverage
make test-coverage
```

### 5. Local Development

```sh
# Create local Kind cluster and deploy
make deploy-kind

# Watch logs
kubectl logs -f -n default deployment/musicservice-controller-manager

# Apply sample MusicService
kubectl apply -f config/samples/musicservice_sample.yaml
```

## Code Guidelines

### Project Structure

```
api/v1/              - CRD definitions
internal/controller/ - Main reconciliation logic
internal/reconciler/ - Domain-specific reconcilers (app, database, storage)
internal/builder/    - Kubernetes resource builders
internal/status/     - Status management
test/                - Tests (e2e, utils)
```

### Coding Standards

- Follow Go best practices and idiomatic patterns
- Write unit tests for new functionality (aim for >80% coverage)
- Add comments for exported functions and types
- Use meaningful variable and function names
- Keep functions focused and small

### Commit Messages

Follow conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `chore`: Maintenance tasks

**Examples:**
```
feat(database): add support for PostgreSQL replication

fix(controller): resolve race condition in status update

docs(readme): update installation instructions
```

## Pull Request Process

1. **Create a branch** from `main`:
   ```sh
   git checkout -b feat/my-new-feature
   ```

2. **Make your changes** and commit:
   ```sh
   git add .
   git commit -m "feat(scope): description"
   ```

3. **Run tests** before pushing:
   ```sh
   make test
   make lint
   ```

4. **Push and create PR**:
   ```sh
   git push origin feat/my-new-feature
   ```

5. **Fill PR template** with:
   - Description of changes
   - Related issues
   - Testing performed
   - Screenshots (if UI changes)

## Testing

### Unit Tests

```sh
# Run all unit tests
make test

# Run specific package
go test ./internal/controller/...

# With coverage
make test-coverage
```

### E2E Tests

```sh
# Run end-to-end tests
make test-e2e

# This will:
# 1. Create Kind cluster
# 2. Build and load operator image
# 3. Deploy operator
# 4. Run test scenarios
# 5. Clean up
```

## Reporting Issues

When reporting bugs, please include:

- **Environment**: OS, Go version, Kubernetes version
- **Steps to reproduce**
- **Expected behavior**
- **Actual behavior**
- **Logs**: Controller logs, kubectl describe output
- **MusicService YAML**: The resource you're trying to deploy

## Feature Requests

When suggesting features:

- **Use case**: Why is this feature needed?
- **Proposed solution**: How should it work?
- **Alternatives**: Other approaches considered
- **Breaking changes**: Impact on existing users

## Code Review Process

All PRs require:
- ✅ Passing CI/CD checks
- ✅ Code review approval
- ✅ Test coverage maintained
- ✅ Documentation updated

## Questions?

- 📖 Read the [README.md](README.md)
- 📚 Check [TESTING_GUIDE.md](TESTING_GUIDE.md)
- 💬 Open a GitHub Discussion
- 🐛 Create an Issue

Thank you for contributing! 🙏
