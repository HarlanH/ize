## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Auto-syncs to JSONL for version control
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**
```bash
bd ready --json
```

**Create new issues:**
```bash
bd create "Issue title" -t bug|feature|task -p 0-4 --json
bd create "Issue title" -p 1 --deps discovered-from:bd-123 --json
```
**Note:** Default priority is P2 (Medium) if not specified.

**Dependency types (IMPORTANT):**
- Valid dependency types are: `blocks`, `related`, `parent-child`, `discovered-from`
- There is **no** `depends-on` dependency type (don’t use it)

**Common patterns:**
```bash
# Epic + child task
bd create "My epic" -t epic -p 1 --json
bd create "My task" -t task -p 1 --deps parent-child:<epic-id> --json

# Sequencing work (A must be done before B)
bd dep add <issue-b> <issue-a> -t blocks --json
```

**Claim and update:**
```bash
bd update bd-42 --status in_progress --json
bd update bd-42 --priority 1 --json
bd update bd-42 --labels "frontend,urgent" --json  # Add/update labels
```

**Search and count:**
```bash
bd search "text query" --json                    # Search issues by text
bd search "query" --priority 1 --json           # With filters
bd count --json                                  # Count and group issues
```

**Complete work:**
```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task**: `bd update <id> --status in_progress`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" -p 1 --deps discovered-from:<parent-id>`
5. **Keep in progress**: Leave the issue in `in_progress` status after implementation. Do NOT close until the user has tested and approved the functionality.
6. **Commit together**: Always commit the `.beads/issues.jsonl` file together with the code changes so issue state stays in sync with code state
7. **User closes**: The user will close the issue after testing/approval, or you may close it only after explicit user approval

### Auto-Sync

bd automatically syncs with git:
- Exports to `.beads/issues.jsonl` after changes (5s debounce)
- Imports from JSONL when newer (e.g., after `git pull`)
- No manual export/import needed!

### MCP Server (Recommended)

If using Claude or MCP-compatible clients, install the beads MCP server:

```bash
pip install beads-mcp
```

Add to MCP config (e.g., `~/.config/claude/config.json`):
```json
{
  "beads": {
    "command": "beads-mcp",
    "args": []
  }
}
```

Then use `mcp__beads__*` functions instead of CLI commands.

### New Features (v0.24+)

**New Commands:**
- `bd search` - Efficient text search across issues with date and priority filters
- `bd count` - Count and group issues for quick statistics
- `bd clean` - Remove temporary merge artifacts

**Enhanced Commands:**
- `bd update` now supports label operations: `bd update <id> --labels "tag1,tag2" --json`
- `bd doctor --fix` - Automatically repair common database issues
- `bd list` accepts both integer (0-4) and P-format (P0-P4) for priority flags
- `bd sync` now auto-resolves conflicts instead of failing

**Performance Improvements:**
- `GetReadyWork` optimization using `blocked_issues_cache` for faster queries
- Bulk label fetching in `bd list` eliminates N+1 query issues

### Known Limitations & Workarounds

**Changing Issue Type:**
- ❌ **Not supported via CLI**: The `bd update` command does not have a flag to change `issue_type`
- ❌ **Not supported via MCP**: The `mcp_beads_update` function does not accept `issue_type` as a parameter
- ✅ **Workaround**: Update the SQLite database directly:
```python
import sqlite3
from datetime import datetime

db_path = '.beads/beads.db'
conn = sqlite3.connect(db_path)
cursor = conn.cursor()

cursor.execute('''
    UPDATE issues
    SET issue_type = ?, updated_at = ?
    WHERE id = ?
''', ('feature', datetime.now().isoformat(), 'issue-id'))

conn.commit()
conn.close()
```
- The database will auto-sync to JSONL after the update

**Priority Parameter Format:**
- When using MCP functions, pass priority as an integer (0-4), not a string like "P1" or "3"
- CLI accepts both formats: `bd update <id> --priority 3` or `bd update <id> --priority P3`

**Database vs JSONL Sync:**
- The database (`.beads/beads.db`) is the source of truth, not the JSONL files
- If you manually edit JSONL, you may need to run `bd import` to sync to database
- If database is newer than JSONL, `bd sync` will export automatically
- If JSONL is newer than database, you'll need to import first: `bd import` (or it will auto-import)
- ✅ **Improved in v0.24+**: `bd sync` now auto-resolves conflicts instead of failing, making sync operations more reliable

**`bd import` Hanging Issues:**
- ⚠️ **Stale lock file**: If `bd import` hangs, check for a stale `.beads/daemon.lock` file. Remove it if the daemon process (PID in lock file) is not running: `rm .beads/daemon.lock`
- ⚠️ **Missing input file**: `bd import` without `-i` reads from stdin and will appear to hang. Always use `-i` flag: `bd import -i .beads/issues.jsonl --json`
- ✅ **Use dry-run first**: Test imports with `--dry-run` to preview changes: `bd import --dry-run -i .beads/issues.jsonl --json`
- ✅ **Improved in v0.24+**: Sandbox escape hatches and daemon stability improvements reduce hanging issues. The daemon also no longer exits when the launcher process exits (fixes macOS issue).

### Managing AI-Generated Planning Documents

AI assistants often create planning and design documents during development:
- PLAN.md, IMPLEMENTATION.md, ARCHITECTURE.md
- DESIGN.md, CODEBASE_SUMMARY.md, INTEGRATION_PLAN.md
- TESTING_GUIDE.md, TECHNICAL_DESIGN.md, and similar files

**Best Practice: Use a dedicated directory for these ephemeral files**

**Recommended approach:**
- Create a `history/` directory in the project root
- Store ALL AI-generated planning/design docs in `history/`
- Keep the repository root clean and focused on permanent project files
- Only access `history/` when explicitly asked to review past planning

**Example .gitignore entry (optional):**
```
# AI planning documents (ephemeral)
history/
```

**Benefits:**
- ✅ Clean repository root
- ✅ Clear separation between ephemeral and permanent documentation
- ✅ Easy to exclude from version control if desired
- ✅ Preserves planning history for archeological research
- ✅ Reduces noise when browsing the project

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ✅ Store AI planning docs in `history/` directory
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems
- ❌ Do NOT clutter repo root with planning documents

For more details, see README.md and QUICKSTART.md.

## Development Guidelines

### Code Standards

**Go:**
- **Module**: Single module `ize` at repository root
- **Package structure**: Standard Go layout with `cmd/` and `internal/` directories
  - `cmd/server/` - Application entrypoints (main.go)
  - `internal/` - Private application code (not importable by other modules)
    - `internal/config/` - Configuration loading and validation
    - `internal/httpapi/` - HTTP handlers and DTOs
    - `internal/algolia/` - Algolia client wrapper and interfaces
    - `internal/ize/` - Core algorithm processing module
- **Error handling**: Use `fmt.Errorf` with `%w` verb for error wrapping, return errors explicitly
- **Testing**: Use standard `testing` package, test files end with `_test.go`
- **Interfaces**: Define interfaces in the consuming package (e.g., `algolia.ClientInterface` in `internal/algolia`)
- **Naming**: 
  - Exported types/functions use PascalCase
  - Unexported use camelCase
  - Constructor functions use `New` prefix (e.g., `NewClient`, `NewSearchHandler`)
- **Documentation**: Export public APIs with doc comments
- **Dependencies**: Keep `go.mod` clean, use `go mod tidy` regularly

**Frontend (Vue.js):**
- **Linting**: ESLint with Vue 3 plugin and Prettier
- **Formatting**: Prettier (via ESLint config)
- **Testing**: Vitest

**General:**
- Pre-commit hooks automatically format code on commit

### File Organization

**Repository Structure:**
```
ize/
├── .beads/              # Beads issue tracker database and config
├── backend/             # Go backend application
│   ├── cmd/
│   │   └── server/     # Main application entrypoint
│   ├── internal/       # Private application packages
│   │   ├── algolia/    # Algolia client wrapper
│   │   ├── config/     # Configuration management
│   │   ├── httpapi/    # HTTP handlers and DTOs
│   │   └── ize/        # Core algorithm processing
│   ├── config.json.example  # Example config file
│   └── go.mod          # Go module definition
├── frontend/            # Vue.js frontend application
│   ├── src/
│   │   ├── api/        # API client functions
│   │   ├── components/ # Vue components
│   │   ├── App.vue     # Root component
│   │   ├── main.ts     # Application entrypoint
│   │   └── types.ts    # TypeScript type definitions
│   ├── package.json    # Node.js dependencies
│   └── vite.config.ts  # Vite configuration
├── history/             # AI-generated planning documents (ephemeral)
├── AGENTS.md           # This file - agent guidelines
├── LICENSE             # MIT license
├── Makefile            # Build automation
└── README.md           # Project documentation
```

**Key Principles:**
- `backend/` and `frontend/` are separate applications in a monorepo
- `internal/` packages are not importable outside the module
- `cmd/` contains application entrypoints
- `history/` stores ephemeral AI planning documents
- Config files with secrets are gitignored (use `.example` files as templates)

### Landing the Plane

When the user says "let's land the plane", follow this clean session-ending protocol:

* File beads issues for any remaining work that needs follow-up
* Ensure all quality gates pass (only if code changes were made) - run tests, linters, builds (update existing P0 issues if broken, create new ones if none exist)
  - Note: Documentation-only changes (e.g., updating AGENTS.md or files in `history/`) do not count as code changes
* Update beads issues - only close issues that have been explicitly tested and approved by the user. Leave others in `in_progress` status.
* Sync the issue tracker carefully - Work methodically to ensure both local and remote issues merge safely. `bd sync` now auto-resolves conflicts (v0.24+), but you may still need to pull, handle conflicts, sync the database, and verify consistency. Be creative and patient - the goal is clean reconciliation where no issues are lost. Use `bd clean` to remove temporary merge artifacts if needed.
* Clean up git state - Clear old stashes and prune dead remote branches:
```bash
git stash clear                    # Remove old stashes
git remote prune origin            # Clean up deleted remote branches
```
* Verify clean state - Ensure all changes are committed and pushed, no untracked files remain
* Choose a follow-up issue for next session - Use `bd ready` to find unblocked work
* Provide a prompt for the user to give to you in the next session
* Format: "Continue work on bd-X: [issue title]. [Brief context about what's been done and what's next]"
