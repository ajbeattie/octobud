# Query Engine

Octobud uses a query language inspired by Gmail to filter and search your notifications. The query engine parses your queries, validates them, and translates them into database searches.

## Technical Overview

The query engine is built as a multi-stage pipeline with the following components:

1. **Lexer** — Tokenizes query strings into individual tokens (fields, operators, values, etc.)
2. **Parser** — Builds an Abstract Syntax Tree (AST) from tokens, handling operator precedence and grouping
3. **Validator** — Validates field names and values against allowed options
4. **SQL Builder** — Generates parameterized SQL queries from the AST, adding JOINs only when needed
5. **Evaluator** — Evaluates queries in-memory for rule matching and action hints

The engine is designed as a pure translation layer—business logic defaults (like inbox filtering) are handled separately. This separation allows the same query parsing logic to be used for both database queries and in-memory evaluation.

**Processing Pipeline:**
```
Query String → Lexer → Parser → AST → Validator → SQL Builder → Database Query
                                  ↓
                              Evaluator → In-Memory Matching
```

**Key Design Principles:**
- **Pure translation** — No business logic in the parser
- **Reusability** — Same parser for SQL generation and in-memory evaluation
- **Efficiency** — JOINs only added when needed, query parsing happens once per request
- **Safety** — Parameterized queries prevent SQL injection

## How Queries Work

### Query Processing

When you enter a query, Octobud:

1. **Parses** the query string into a structured representation
2. **Validates** that all fields and values are valid
3. **Translates** the query into database filters
4. **Applies defaults** based on context (e.g., inbox excludes archived/muted items)
5. **Executes** the search and returns matching notifications

### Query Components

**Fields and Filters**
- Field filters like `repo:`, `type:`, `author:` search specific notification properties
- Special operators like `in:` and `is:` provide shortcuts for common states
- Free text searches across multiple fields (title, repository, author, etc.)

**Logical Operators**
- `AND` (implicit when space-separated) — All conditions must match
- `OR` — Any condition can match
- `NOT` or `-` — Exclude matching items
- Parentheses `()` — Group conditions to control precedence

**Contains Matching**
- All search values use contains matching, not exact matching
- `repo:traefik` matches `traefik/traefik` and `traefik/other`
- `author:bot` matches `dependabot` and `github-actions[bot]`

### Default Behavior

Queries have smart defaults based on context:

- **Empty query** → Shows inbox (excludes archived, snoozed, muted, filtered)
- **Queries with `in:` operator** → No defaults applied (you're explicitly specifying location)
- **Other queries** → Excludes muted notifications unless explicitly requested

### Query Evaluation

The query engine evaluates notifications by:

- Checking if the notification matches each filter condition
- Combining results using logical operators (AND, OR, NOT)
- Respecting operator precedence (parentheses → NOT → AND → OR)
- Applying the query's default filters where appropriate

This evaluation happens both for:
- **Database queries** — Filtering notifications when fetching from the database
- **In-memory matching** — Checking if notifications match rules or views (e.g., for action hints)

## Query Features

### Gmail-Style Syntax

Octobud's query language follows familiar patterns from Gmail:

- `in:inbox`, `in:archive` for location
- `is:read`, `is:unread` for state
- Boolean operators and grouping with parentheses
- Comma-separated values for OR (`repo:cli,other`)

### Validation and Error Messages

The query engine validates your queries and provides helpful error messages for:

- Unknown fields (e.g., `badfield:value`)
- Invalid values for operators (e.g., `in:badvalue`)
- Syntax errors (e.g., mismatched parentheses)
- Unclosed quotes

### Performance

- Queries are efficiently translated to database searches
- Only necessary data is joined (e.g., repository info when filtering by repo)
- Complex queries with multiple filters remain fast
- Query parsing happens once per request, not per notification

