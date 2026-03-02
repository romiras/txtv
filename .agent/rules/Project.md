# Project-Specific Rules for `txtv`

## Project Infrastructure

- **Build System**: The project uses **GNU Make**. Always use the `Makefile` for building, testing, and formatting.
- **Testing Framework**: Standard **Go `testing` package**. Run tests using `make test`.

## Ticket & Git Lifecycle

Follow this lifecycle for every feature, bug fix, or chore:

1.  **Ticket Creation**:
    - Create a ticket using `tk create "Title" -t <type> -d "Description"`.
    - Capture the generated Ticket ID (e.g., `txt-xugl`).
2.  **Branching**:
    - Create and switch to a feature branch derived from the Ticket ID: `git checkout -b <ticket-id>`.
3.  **Implementation & Verification**:
    - Keep changes focused on the ticket objective.
    - Run tests frequently: `make test`.
    - Ensure code is formatted: `make fmt`.
4.  **Finalization**:
    - Commit changes with a descriptive message referencing the ticket: `git commit -m "type(scope): message. Resolves ticket <id>."`.
    - Switch to `master` and merge the branch: `git checkout master && git merge <id>`.
    - **Delete the branch** immediately after merging: `git branch -d <id>`.
    - **Close the ticket** using `tk close <id>`.

## File & Dependency Constraints

- **File Restrictions**:
  - Do NOT modify any files in the `.git/` directory.
  - Do NOT modify `docs/` or specifications unless explicitly instructed.
- **Dependency Policy**:
  - Avoid adding new external dependencies unless strictly required by the specification (e.g., `golang.org/x/text/segment`).
- **Git Ignoring**:
  - **NEVER** add `docs/` to `.gitignore`. Documentation must be version-controlled.
  - Use `/` prefix in `.gitignore` for root-level binaries (e.g., `/txtv`) to avoid unintentionally ignoring entire directories.

## Learned Lessons & Best Practices

- **Project Structure**: Strictly follow the standard Go project layout:
  - Entry points in `cmd/<app>/main.go`.
  - Core logic and internal packages in `internal/`.
- **Testing Requirements**:
  - Every new logical layer (Engine, Segmenter, etc.) must have a corresponding `_test.go` file in its package.
  - Test suites must pass byte-for-byte correctness checks for pass-through behavior.
- **Engine Logic**:
  - Maintain $O(1)$ memory usage throughout the read/segment/write loop.
  - Always implement the 1MB "soft stop" fail-safe.
