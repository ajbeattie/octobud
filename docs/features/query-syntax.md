# Query Syntax

Octobud uses a query language to filter notifications. You can use queries in views, rules, and the search bar.

## Basic Syntax

Queries are made up of filters separated by spaces (implicit AND):

```
repo:owner/name is:unread type:PullRequest
```

**Important:** All search terms and values in key-value pairs use **contains matching** (not exact matching). For example, `repo:traefik` will match any repository full name that contains "traefik", such as `traefik/traefik` or `traefik/other`.

## Available Filters

### Status Filters (`is:`)

| Filter | Description |
|--------|-------------|
| `is:read` | Read notifications |
| `is:unread` | Unread notifications |
| `is:starred` | Starred notifications |
| `is:snoozed` | Currently snoozed notifications |
| `is:archived` | Archived notifications |
| `is:muted` | Muted notifications |
| `is:filtered` | Filtered (skipped inbox) notifications |

### Location Filters (`in:`)

| Filter | Description |
|--------|-------------|
| `in:inbox` | In inbox (not archived, snoozed, muted, or filtered) |
| `in:archive` | In archive (not muted) |
| `in:snoozed` | Currently snoozed (not archived or muted) |
| `in:filtered` | Filtered/skipped inbox (notifications that were automatically filtered by rules and skipped the inbox) |
| `in:anywhere` | All notifications (no location filter) |

**Note:** Use `in:inbox,filtered` (equivalent to `in:inbox OR in:filtered`) to include notifications that have skipped the inbox due to rule automation. `in:inbox` queries only search the inbox, so filtered notifications won't appear unless you explicitly include them with `in:filtered` or `in:anywhere`.

### Type Filters (`type:`)

| Filter | Description |
|--------|-------------|
| `type:PullRequest` | Pull request notifications |
| `type:Issue` | Issue notifications |
| `type:Release` | Release notifications |
| `type:Discussion` | Discussion notifications |
| `type:Commit` | Commit notifications |
| `type:CheckSuite` | CI/CD check suite notifications |
| `type:RepositoryVulnerabilityAlert` | Security alert notifications |

### Reason Filters (`reason:`)

| Filter | Description |
|--------|-------------|
| `reason:review_requested` | Review was requested |
| `reason:mention` | You were mentioned |
| `reason:author` | You authored the item |
| `reason:comment` | Someone commented |
| `reason:assign` | You were assigned |
| `reason:team_mention` | Your team was mentioned |
| `reason:subscribed` | You're subscribed |
| `reason:state_change` | State changed |
| `reason:ci_activity` | CI activity |

### Repository Filters

| Filter | Description |
|--------|-------------|
| `repo:owner/name` | Match repository (contains matching - e.g., `repo:traefik` matches `traefik/traefik` or `traefik/other`) |
| `org:owner` | All repos in an organization (contains matching - matches owner/*) |

### Author Filter

| Filter | Description |
|--------|-------------|
| `author:username` | Filter by author (contains matching) |

### State Filters

| Filter | Description |
|--------|-------------|
| `state:open` | Open issues/PRs |
| `state:closed` | Closed issues/PRs |
| `merged:true` | Merged pull requests |
| `merged:false` | Unmerged pull requests |
| `state_reason:completed` | Issues closed as completed |
| `state_reason:not_planned` | Issues closed as not planned |

### Tag Filters

| Filter | Description |
|--------|-------------|
| `tags:tag-name` | Has tag matching pattern (contains matching) |

### Boolean Filters

These can be used with values `true`/`false`, `yes`/`no`, or `1`/`0`:

| Filter | Description |
|--------|-------------|
| `read:true` | Read notifications |
| `archived:true` | Archived notifications |
| `muted:true` | Muted notifications |
| `snoozed:true` | Currently snoozed |
| `filtered:true` | Filtered notifications |

### Free Text Search

Any text without a field prefix searches across (using contains matching):
- Notification title
- Repository name
- Author
- Subject type
- Subject state
- PR/Issue number

```
dependabot          # Matches title, author, etc. (contains)
fix bug             # Multiple words (AND, both contains)
```

## Negation

Prefix any filter with `-` to negate it:

```
-is:read           # Not read (unread)
-type:Issue        # Not an issue
-repo:owner/name   # Not from this repo
-author:dependabot # Not from dependabot
```

## Boolean Operators

### AND (implicit)

Filters separated by spaces are combined with AND:

```
is:unread type:PullRequest
# Unread AND pull request
```

### AND (explicit)

```
is:unread AND type:PullRequest
```

### OR

Use `OR` between filters:

```
type:PullRequest OR type:Issue
# Pull requests OR issues
```

### Grouping

Use parentheses for complex queries:

```
(type:PullRequest OR type:Issue) is:unread
# (PR or issue) AND unread
```

## Examples

### Unread PR reviews

```
is:unread type:PullRequest reason:review_requested
```

### My mentions across all repos

```
reason:mention in:anywhere
```

### Issues in a specific org

```
org:my-company type:Issue is:unread
```

Note: `org:my-company` uses contains matching, so it will match any organization name containing "my-company".

### Everything except CI notifications

```
-reason:ci_activity
```

### Starred but not archived

```
is:starred -is:archived
```

### Open PRs that need review

```
type:PullRequest reason:review_requested state:open -is:archived
```

### Merged PRs

```
type:PullRequest merged:true
```

### Dependabot PRs to archive

```
author:dependabot type:PullRequest
```

Note: `author:dependabot` uses contains matching, so it will match authors like "dependabot" or "dependabot[bot]".

### Notifications with a specific tag

```
tags:urgent
```

### Filtered notifications (skipped inbox)

```
in:filtered
```

Shows notifications that were automatically filtered by rules and skipped the inbox. This is useful for reviewing what rules have filtered and ensuring important notifications aren't being hidden.

```
in:filtered type:PullRequest
```

Shows filtered pull request notifications.

```
in:anywhere is:filtered
```

Shows all filtered notifications regardless of their location (alternative syntax using `is:filtered`).

## Tips

1. **Start Simple** — Begin with one or two filters and add more as needed
2. **Use Views** — Save complex queries as views for quick access
3. **Try Negation** — Sometimes it's easier to exclude what you don't want
4. **Combine with Rules** — Use queries in rules to auto-organize notifications
5. **Review Filtered** — Periodically check `in:filtered` to ensure rules aren't hiding important notifications
6. **Contains Matching** — All search terms and values use contains matching, not exact matching. For example, `repo:traefik` matches `traefik/traefik` or `traefik/other`, and `author:bot` matches `dependabot` or `github-actions[bot]`
