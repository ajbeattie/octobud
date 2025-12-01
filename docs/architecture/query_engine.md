# Query Parser

A query parser and SQL generator for notification filtering, supporting Gmail-style syntax with logical operators (AND, OR, NOT) and parentheses grouping.

## Features

- **Gmail-inspired syntax** with `in:` and `is:` operators
- **Logical operators**: AND, OR, NOT with proper precedence
- **Parentheses grouping** for complex expressions
- **Comma-separated OR** within fields (`repo:cli,other`)
- **Free text search** across multiple fields
- **Full validation** with helpful error messages
- **Pure translation layer** - business logic defaults handled by caller

## Query Syntax

### Basic Field Filtering

```
repo:cli                    # Repository contains "cli"
reason:review_requested     # Notification reason
type:PullRequest            # Subject type
author:username             # Author login
state:open                  # PR/Issue state
tags:urgent                 # Tag slug contains "urgent" (partial match)
```

### Special Operators

#### `in:` Operator (Location)
```
in:inbox       # Not archived, snoozed, filtered, or muted
in:archive     # Archived notifications (excluding muted)
in:snoozed     # Snoozed notifications (excluding archived/muted)
in:filtered    # Filtered by rules (auto-categorized)
in:anywhere    # All notifications including muted
```

#### `is:` Operator (State)
```
is:read        # Read notifications
is:unread      # Unread notifications  
is:archived    # Archived notifications
is:muted       # Muted notifications
is:snoozed     # Snoozed notifications
is:filtered    # Filtered by rules
```

### Boolean Fields
```
read:true           # Explicitly read
archived:false      # Not archived
muted:true          # Muted
snoozed:false       # Not snoozed
filtered:true       # Filtered by rules
```

### Logical Operators

```
repo:cli AND is:unread              # Must match both
repo:cli OR repo:other              # Match either
NOT author:bot                      # Exclude bots
repo:cli is:unread                  # Implicit AND (whitespace)
```

**Precedence** (highest to lowest):
1. Parentheses `()`
2. `NOT`
3. `AND`
4. `OR`

### Grouping

```
(repo:cli OR repo:other) AND is:unread
NOT (in:archive OR in:snoozed)
((a AND b) OR (c AND d)) AND e
```

### Comma-Separated OR

```
repo:cli,other              # repo:cli OR repo:other
reason:mention,review       # Multiple reasons
tags:urgent,bug             # Tag slug contains "urgent" OR "bug"
```

### Free Text Search

Words without a colon are searched across title, type, repo, author, and state:

```
urgent fix                          # Free text search
repo:cli urgent                     # Combined with filters
"fix: memory leak"                  # Quoted phrase
```

### Quoted Strings

```
subject:"urgent fix"
repo:"my-org/my-repo"
```

## Usage

### Basic Query Building

```go
import "github.com/ajbeattie/octobud/backend/internal/query"

// Parse and build SQL (no defaults applied)
q, err := query.BuildQuery("repo:cli is:unread", 50, 0)
if err != nil {
    // Handle error
}

// q.Joins contains JOIN clauses
// q.Where contains WHERE conditions
// q.Args contains query parameters
// q.Limit and q.Offset are set
```

### With Inbox Defaults

```go
// Build query with unified defaults (excludes archived, snoozed, filtered, muted for empty query)
// Defaults are NOT applied if query contains explicit `in:` operator
q, err := query.BuildQuery("repo:cli", 50, 0)
```

### Advanced: AST Manipulation

```go
// Parse and validate
ast, err := query.ParseAndValidate("repo:cli AND is:unread")
if err != nil {
    // Handle parse/validation error
}

// Build SQL from AST
builder := query.NewSQLBuilder()
q, err := builder.Build(ast)

// Apply custom defaults
q = query.ApplyInboxDefaults(q)
```

## Examples

### Simple Filtering
```
repo:cli is:unread
→ Shows unread notifications from cli repo

reason:review_requested,mention
→ Shows review requests OR mentions

type:PullRequest state:open
→ Shows open pull requests
```

### View-Based Queries
```
in:inbox repo:cli
→ Inbox items from cli repo

in:snoozed
→ All snoozed notifications

in:archive
→ All archived notifications

in:anywhere repo:cli/cli
→ All notifications from cli/cli (including muted)
```

### Complex Filtering
```
(repo:cli OR repo:other) AND is:unread
→ Unread notifications from either repo

repo:cli reason:review_requested,mention is:unread
→ Unread review requests or mentions from cli

NOT (author:bot OR author:dependabot)
→ Exclude automated authors

tags:urgent AND NOT tags:resolved
→ Tagged as urgent but not resolved
```

### With Free Text
```
repo:cli urgent
→ Notifications from cli containing "urgent"

"fix: memory leak" repo:cli
→ Specific phrase in cli repo
```

### Very Complex
```
((repo:cli/cli AND is:unread) OR (in:snoozed AND repo:github/docs)) AND NOT author:bot
→ (Unread cli items OR snoozed docs items) excluding bots

(reason:review_requested OR reason:mention) AND is:unread AND repo:cli,other
→ Unread review requests or mentions from cli or other repo
```

## Architecture

### Components

1. **Lexer** (`lexer.go`) - Tokenizes query strings
2. **Parser** (`parser.go`) - Builds AST with proper precedence
3. **AST** (`ast.go`) - Node types representing query structure
4. **SQL Builder** (`sql_builder.go`) - Generates parameterized SQL
5. **Validator** (`validator.go`) - Validates field names and values
6. **Public API** (`query.go`) - Convenient entry points

### Design Principles

- **Pure translation** - Parser doesn't apply business logic
- **Separation of concerns** - Defaults handled by caller
- **Explicit over implicit** - Clear what SQL is generated
- **Testable** - Each component tested in isolation
- **Extensible** - Easy to add new fields or operators

## Supported Fields

| Field | Description | Example |
|-------|-------------|---------|
| `in` | Location (inbox, archive, snoozed, anywhere) | `in:inbox` |
| `is` | State (read, unread, archived, muted, snoozed) | `is:unread` |
| `repo` / `repository` | Repository full name (contains match) | `repo:cli` |
| `org` | Organization (prefix match) | `org:github` |
| `reason` | Notification reason | `reason:review_requested` |
| `type` / `subject_type` | Subject type | `type:PullRequest` |
| `author` | Author login | `author:username` |
| `state` | PR/Issue state | `state:open` |
| `read` | Read status (boolean) | `read:true` |
| `archived` | Archived status (boolean) | `archived:false` |
| `muted` | Muted status (boolean) | `muted:true` |
| `snoozed` | Snoozed status (boolean) | `snoozed:false` |
| `filtered` | Filtered status (boolean) | `filtered:true` |
| `tags` | Tags assigned to notification | `tags:urgent` |

## Error Handling

The parser provides detailed error messages:

```go
_, err := query.BuildQuery("badfield:value", 50, 0)
// Error: "validation error: validation errors: unknown field: badfield"

_, err := query.BuildQuery("in:badvalue", 50, 0)
// Error: "validation error: validation errors: invalid value for in: operator: badvalue (valid: inbox, archive, snoozed, anywhere)"

_, err := query.BuildQuery("(repo:cli", 50, 0)  
// Error: "parse error: expected ')' at position X, got EOF"
```

## Testing

The package includes tests:

- **Lexer tests** - Token generation
- **Parser tests** - AST construction, precedence, grouping
- **SQL builder tests** - SQL generation for all features
- **Integration tests** - End-to-end query building
- **Validation tests** - Error cases

Run tests:
```bash
go test ./internal/query -v
```

## Future Enhancements

Potential additions (not currently implemented):

- Explicit `AND`/`OR` within values: `repo:(cli OR other)`
- Date operators: `after:`, `before:`, `newer_than:`, `older_than:`
- `has:` operator for special content
- Negation within comma lists: `repo:cli,-cli/cli` (exclude subdirectory)

## Performance

- Generates parameterized queries ($1, $2, etc.) for SQL injection safety
- Only adds JOINs when needed (repo, PR fields)
- Efficient AST walking with single pass
- No regex compilation or heavy parsing overhead

