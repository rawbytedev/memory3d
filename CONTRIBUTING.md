# Contributing to Memory3D

Thank you for interest in contributing to Memory3D! This document provides guidelines for contributing to the project.

## Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [Getting Started](#getting-started)
3. [Development Setup](#development-setup)
4. [Making Changes](#making-changes)
5. [Testing](#testing)
6. [Documentation](#documentation)
7. [Submitting Changes](#submitting-changes)
8. [Coding Standards](#coding-standards)
9. [Commit Messages](#commit-messages)
10. [Pull Request Process](#pull-request-process)

## Code of Conduct

### Our Pledge

We are committed to providing a welcoming and inspiring community for all. Please read and abide by our Code of Conduct:

- **Be Respectful**: Treat all contributors with respect and dignity
- **Be Inclusive**: Welcome diverse perspectives and backgrounds
- **Be Constructive**: Provide helpful feedback and focus on solutions
- **Be Professional**: Keep discussions on-topic and productive

### Enforcement

Violations of the Code of Conduct may result in removal from the project. Please report violations to the project maintainers.

## Getting Started

### Prerequisites

- Go 1.25.2 or later
- Git
- Basic knowledge of Go programming
- Familiarity with version control

### Fork and Clone

```bash
# Fork the repository on GitHub
# Clone your fork
git clone https://github.com/YOUR_USERNAME/memory3d.git
cd memory3d

# Add upstream remote
git remote add upstream https://github.com/rawbytedev/memory3d.git
```

### Create a Branch

```bash
# Update from upstream
git fetch upstream
git rebase upstream/main

# Create feature branch
git checkout -b feature/your-feature-name
```

## Development Setup

### Install Dependencies

```bash
go mod download
go mod tidy
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test -v ./internal/allocator/...
```

### Run Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./test/benchmarks/...

# Run specific benchmark
go test -bench=BenchmarkAllocateSmall -benchtime=5x ./test/benchmarks/...

# Get memory profile
go test -bench=. -memprofile=mem.prof ./test/benchmarks/...
go tool pprof mem.prof
```

### Build Project

```bash
# Build main binary
go build -o vm3d ./cmd/vm3d/main.go

# Build with optimizations
go build -ldflags="-s -w" -o vm3d ./cmd/vm3d/main.go
```

## Making Changes

### Code Organization

Memory3D is organized as follows:

```arch
memory3d/
├── cmd/
│   └── vm3d/               # Command-line applications
│       ├── main.go
│       └── cli/
├── internal/
│   ├── vm/                 # Virtual machine core
│   ├── allocator/          # Memory allocator
│   ├── types/              # Type definitions
│   ├── merkle/             # Merkle tree
│   └── cache/              # Caching layer
├── test/
│   ├── unit/               # Unit tests
│   ├── integration/        # Integration tests
│   └── benchmarks/         # Performance benchmarks
├── docs/                   # Documentation
└── examples/               # Example programs
```

### What to Contribute

**Good Contribution Areas:**

- Bug fixes with test cases
- Performance improvements with benchmarks
- Documentation improvements
- New features with design discussion
- Test coverage expansion
- Error handling improvements

**Before You Start:**

1. Check [Issues](https://github.com/rawbytedev/memory3d/issues) to avoid duplicates
2. Open an issue to discuss major features
3. Reference existing issues in your PR

### Code Style

Follow Go conventions:

```bash
# Format code
gofmt -w ./...

# Lint code
go vet ./...

# Use go fmt standards
go fmt ./...
```

**Style Guidelines:**

- Use `camelCase` for function names (except exported functions)
- Use `PascalCase` for exported functions and types
- Keep lines under 100 characters where reasonable
- Add comments for exported functions
- Use short variable names in small scopes (i, j, k) but descriptive elsewhere

### Example Code Contribution

```go
// ✓ Good
func (a *Allocator3D) allocateInRegion(size uint32, region *Region) (*Allocation, error) {
    if size == 0 {
        return nil, fmt.Errorf("allocation size must be > 0")
    }
    // Implementation...
    return alloc, nil
}

// ✗ Bad
func (a *Allocator3D) AllocInRegion(s uint32, r *Region) (*Allocation, error) {
    // No validation
    // Missing comment
    return alloc, nil
}
```

## Testing

### Writing Tests

All code changes should include tests:

```go
func TestAllocatorAllocate(t *testing.T) {
    config := allocator.Config{...}
    alloc := allocator.NewAllocator3D(config)

    // Test successful allocation
    allocation, err := alloc.Allocate(256, types.RegionTypeHeap, 0)
    assert.NoError(t, err)
    assert.NotNil(t, allocation)
    assert.Equal(t, uint32(256), allocation.Size)

    // Test allocation failure
    _, err = alloc.Allocate(0, types.RegionTypeHeap, 0)
    assert.Error(t, err)
}
```

### Test Coverage

```bash
# Check coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Running Specific Tests

```bash
# Run single test
go test -run TestAllocatorAllocate ./internal/allocator/...

# Run tests matching pattern
go test -run TestAllocator* ./internal/allocator/...

# Run tests excluding pattern
go test -run '^((?!Benchmark).)*$' ./...
```

## Documentation

### Documentation Requirements

All PRs should include:

1. **Updated README.md** if behavior changes
2. **Updated API.md** for API changes
3. **Code comments** for complex logic
4. **Example code** for new features
5. **Changelog entry** (if applicable)

### Writing Documentation

```go
// Good: Clear, concise, explains purpose
// allocateInRegion attempts to allocate memory within an existing region.
// If the region lacks sufficient contiguous space, it returns an error.
func (a *Allocator3D) allocateInRegion(size uint32, region *Region) (*Allocation, error) {}

// Bad: Unclear or missing
func (a *Allocator3D) allocateInRegion(size uint32, region *Region) (*Allocation, error) {}
```

### Documentation Format

- Use Markdown for documentation files
- Include examples for new features
- Update table of contents in long documents
- Link to related sections
- Add diagrams for complex concepts

## Submitting Changes

### Before You Push

1. **Run tests**: `go test ./...`
2. **Run benchmarks**: `go test -bench=. ./test/benchmarks/...`
3. **Format code**: `gofmt -w ./...`
4. **Lint code**: `go vet ./...`
5. **Review your changes**: Check git diff

### Push to Your Fork

```bash
git add .
git commit -m "Add feature: description"
git push origin feature/your-feature-name
```

### Create Pull Request

Go to GitHub and create a PR from your fork to upstream.

## Coding Standards

### Error Handling

```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to allocate memory: %w", err)
}

// Bad: Lose context
if err != nil {
    return err
}
```

### Concurrency

```go
// Use sync.RWMutex for shared state
type VM3D struct {
    mu sync.RWMutex
    state interface{}
}

// Use atomic operations for counters
atomic.AddUint64(&counter, 1)

// Avoid data races: run go test -race ./...
```

### Comments

```go
// Good: Explain WHY, not WHAT
// LRU cache is checked first because most allocations reuse recent regions
func (a *Allocator3D) allocateFromLRU() (*Allocation, error) {}

// Bad: Explains WHAT (obvious from code)
// Get allocation from LRU cache
func (a *Allocator3D) allocateFromLRU() (*Allocation, error) {}
```

### Error Types

```go
// Define error variables for consistency
var (
    ErrOutOfMemory = fmt.Errorf("out of memory")
    ErrInvalidAddress = fmt.Errorf("invalid address")
)

// Use them consistently
if size > available {
    return ErrOutOfMemory
}
```

## Commit Messages

### Format

```txt
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat**: New feature
- **fix**: Bug fix
- **refactor**: Code refactoring
- **perf**: Performance improvement
- **test**: Adding or updating tests
- **docs**: Documentation changes
- **style**: Code style changes
- **chore**: Maintenance tasks

### Examples

```txt
feat(allocator): implement Y-promotion strategy

Add intelligent Y-promotion to consolidate allocations
and reduce fragmentation. Includes configuration options
and gas refund mechanism.

Fixes #42
```

```txt
fix(vm): deadlock in tryFastAllocation

Release read lock before acquiring write lock in LRU
promotion to prevent deadlock.

Fixes #35
```

```txt
docs: update API reference

Add new methods documentation and fix examples in
API.md.
```

## Pull Request Process

### Before Submission

- Code follows style guidelines
- Tests pass: `go test ./...`
- Benchmarks show no regression (Unless a they is a good reason for that)
- Documentation is updated
- Commit messages are clear
- No merge conflicts with main branch

### During Review

- Respond to feedback constructively
- Request review from maintainers
- Be patient with review process
- Make requested changes promptly

### After Approval

A maintainer will merge your PR. Thank you for your contribution!

### After Merge

- Delete your feature branch: `git branch -d feature/your-feature-name`
- Pull latest from upstream: `git pull upstream main`

## Performance Considerations

When submitting performance-related changes:

1. **Provide benchmarks**: Show before/after performance
2. **Test on target platforms**: Windows, Linux, macOS
3. **Profile your changes**: Use `pprof` when possible
4. **Document trade-offs**: Explain memory vs. speed trade-offs

Example benchmark results:

```txt
Before:
BenchmarkAllocateSmall-4    500000    2300 ns/op

After:
BenchmarkAllocateSmall-4    750000    1540 ns/op

Improvement: ~33% faster
```

## Questions?

- Check [Issues](https://github.com/rawbytedev/memory3d/issues) for similar questions
- Open a new issue with `[QUESTION]` prefix
- Read existing documentation thoroughly first

Thank you for making Memory3D better!

---

**Happy contribution!**

For more information, see the [main README](./README.md) or [Architecture guide](./ARCHITECTURE.md).
