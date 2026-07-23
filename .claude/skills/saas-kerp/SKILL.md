```markdown
# saas-kerp Development Patterns

> Auto-generated skill from repository analysis

## Overview
This skill teaches the core development patterns and conventions used in the `saas-kerp` Go codebase. It covers file organization, code style, commit message conventions, and testing approaches, providing practical examples and command suggestions to streamline common workflows.

## Coding Conventions

### File Naming
- Use **snake_case** for all file names.
  - Example:  
    ```plaintext
    user_service.go
    order_handler.go
    ```

### Import Style
- Use **relative imports** within the project.
  - Example:
    ```go
    import (
        "fmt"
        "../models"
        "./utils"
    )
    ```

### Export Style
- Use **named exports** for functions, types, and variables.
  - Example:
    ```go
    // In user_service.go
    package service

    func CreateUser(name string) error {
        // implementation
    }
    ```

### Commit Message Conventions
- Use **conventional commit** style.
- Prefix commit messages with the type, e.g., `fix`.
- Average commit message length: ~63 characters.
  - Example:
    ```
    fix: correct user authentication logic to handle token expiry
    ```

## Workflows

### Fix a Bug
**Trigger:** When a bug is identified and needs to be resolved  
**Command:** `/fix-bug`

1. Identify and reproduce the bug.
2. Create a new branch for the fix.
3. Apply code changes following the coding conventions.
4. Write or update tests to cover the bug scenario.
5. Commit changes using the `fix:` prefix in the commit message.
6. Push the branch and create a pull request for review.

### Add a New Feature
**Trigger:** When implementing a new feature or module  
**Command:** `/add-feature`

1. Plan the feature and determine affected modules.
2. Create a new branch for the feature.
3. Add new files using snake_case naming.
4. Use relative imports for any internal dependencies.
5. Export new functions/types using named exports.
6. Write or update tests for the new feature.
7. Commit changes with a descriptive message (e.g., `feat:` if following extended conventional commits).
8. Push the branch and open a pull request.

### Run Tests
**Trigger:** Before merging changes or verifying functionality  
**Command:** `/run-tests`

1. Locate test files matching the `*.test.*` pattern.
2. Use the appropriate Go test command (e.g., `go test`) to execute tests.
3. Review test results and fix any failing cases.

## Testing Patterns

- Test files follow the `*.test.*` naming pattern.
  - Example:  
    ```plaintext
    user_service.test.go
    ```
- Testing framework is unspecified; use standard Go testing tools unless otherwise noted.
- Place test files alongside the code they test.

  Example test file:
  ```go
  // user_service.test.go
  package service

  import "testing"

  func TestCreateUser(t *testing.T) {
      err := CreateUser("Alice")
      if err != nil {
          t.Errorf("expected no error, got %v", err)
      }
  }
  ```

## Commands
| Command      | Purpose                                  |
|--------------|------------------------------------------|
| /fix-bug     | Start a workflow to fix a bug            |
| /add-feature | Begin implementing a new feature         |
| /run-tests   | Run all tests in the codebase            |
```
