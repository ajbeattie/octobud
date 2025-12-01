# Basic Usage Guide

This guide provides a high-level overview of Octobud's core features. For detailed information, see the linked documentation.

## 1. Actions and Inbox/Triage States

Octobud helps you manage notifications through a set of actions that move them between different states.

### Available Actions

| Action | Keyboard Shortcut | Description |
|--------|------------------|-------------|
| **Star** | `s` | Mark as important - appears in "Starred" view |
| **Mark Read/Unread** | `r` | Toggle read status - unread items stand out |
| **Archive/Unarchive** | `e` | Toggle archive status - move out of inbox or bring back (works for individual notifications) |
| **Unarchive** | `Shift + e` | Unarchive selected notifications (multiselect mode only) |
| **Unstar** | `Shift + s` | Unstar selected notifications (multiselect mode only) |
| **Snooze** | `z` | Hide until a specific date/time - returns automatically |
| **Mute** | `m` | Permanently hide - only visible in "Everything" view |
| **Tag** | `t` | Add custom tags for organization |
| **Allow back into inbox** | `i` | Remove from filtered state (if skipped by rule) |

### Notification States

Notifications move through different states based on your actions:

- **Inbox** — Active notifications that haven't been archived, muted, snoozed, or filtered by rules
- **Archived** — Removed from inbox but still searchable and accessible
- **Snoozed** — Temporarily hidden until a specific time, then automatically return to inbox
- **Muted** — Permanently hidden (only visible in "Everything" view)
- **Filtered** — Skipped inbox due to a rule action but still accessible in custom views
- **Read/Unread** — Read status independent of other states

**Key concept:** Most actions work independently. For example, a notification can be:
- Archived AND unread
- Starred AND snoozed
- Tagged AND muted

### Triage Workflow

A typical workflow:
1. Review inbox for unread notifications
2. **Star** (`s`) important items for follow-up
3. **Archive** (`e`) items you've handled
4. **Snooze** (`z`) items to revisit later
5. **Tag** (`t`) items to categorize (e.g., "followup", "review")

Actions can be applied individually or in bulk using multiselect mode (`v` to toggle).

---

## 2. Query and View Basics

### What are Views?

Views are saved queries that create filtered lists of notifications. They appear in your sidebar and provide quick access to specific subsets of your notifications.

### Built-in Views

Octobud includes several default views:

- **Inbox** — All active notifications (not archived, muted, snoozed, or filtered)
- **Starred** — All starred notifications
- **Archived** — All archived notifications
- **Everything** — All notifications, including muted ones

### Creating a View

1. Type a query in the search bar at the top (e.g., `type:pullrequest reason:review_requested`)
2. Click **Save** to create a view from the current query
3. Or click the **+** button in the sidebar to create a new view
4. Give it a name, choose an icon, and save

### Query Basics

Queries filter notifications using a simple syntax. You can combine multiple filters:

**Simple queries:**
```
type:pullrequest              # Show only pull requests
is:unread                     # Show only unread items
repo:octobud                  # Show notifications from a repo
```

**Combined queries:**
```
type:pullrequest reason:review_requested is:unread
```

This finds: unread pull request review requests.

**Query operators:**
- Space = AND (all conditions must match)
- `OR` = OR (any condition can match)
- `NOT` or `-` = NOT (exclude matching items)
- `()` = Group conditions

**Example:**
```
(reason:mention OR reason:review_requested) AND is:unread
```

This finds: unread notifications that are either mentions or review requests.

### Common Query Filters

- **Status:** `is:read`, `is:unread`, `is:starred`, `is:archived`, `is:snoozed`, `is:muted`
- **Type:** `type:PullRequest`, `type:Issue`, `type:Release`, `type:Discussion`
- **Reason:** `reason:review_requested`, `reason:mention`, `reason:author`, `reason:comment`
- **Repository:** `repo:owner/name` (partial matching supported)
- **Organization:** `org:name` (partial matching supported)
- **Tags:** `tags:name` (partial matching supported)

**Learn more:** See the [Query Syntax Guide](features/query-syntax.md) for complete documentation.

---

## 3. Tag Basics

Tags are custom labels you can assign to notifications to organize them beyond the built-in states.

### Creating Tags

1. Go to **Settings** → **Tags**
2. Click **Add Tag**
3. Enter a name and choose a color
4. Click **Save**

### Using Tags

**Manual tagging:**
- Select a notification and press `t` to open the tag dropdown
- Check tags to add, uncheck to remove
- Works on single notifications or bulk selections

**Rule-based tagging:**
- Create rules that automatically assign tags to matching notifications
- Useful for consistent categorization

**Filtering by tags:**
- Use `tags:name` in queries to filter notifications
- Supports partial matching (e.g., `tags:urg` matches "urgent")
- Combine with other filters: `tags:review is:unread`

### Tag Examples

- `urgent` — Needs immediate attention
- `waiting` — Waiting for someone else
- `review` — Needs your review
- `followup` — Follow up later
- `security` — Security-related notifications

**Learn more:** Tags are covered in the [Views and Rules Guide](features/views-and-rules.md#tags).

---

## 4. Rule Basics

Rules automatically apply actions to notifications that match a query. They help automate your workflow by handling routine tasks.

### How Rules Work

When a new notification arrives, Octobud checks it against all enabled rules in order. If a notification matches a rule's query, the rule's actions are applied automatically.

**Example:** A rule that matches `author:dependabot` can automatically archive all Dependabot notifications, keeping your inbox clean.

### Rule Types

Rules can be **query-based** or **view-linked**:

**Query-based rules:**
- Have their own independent query
- Use when you want a rule that doesn't correspond to any view
- Example: Auto-archive all Dependabot PRs

**View-linked rules:**
- Link to an existing view's query
- Automatically update when you change the view's query
- Use when you want a view and rule to stay in sync
- Example: Auto-star all notifications in your "Needs Review" view

### Available Rule Actions

- **Skip Inbox** — Mark as filtered (won't appear in inbox, but accessible in custom views)
- **Mark as Read** — Automatically mark as read
- **Archive** — Move to archive immediately
- **Star** — Add star automatically
- **Mute** — Permanently hide (also archives)
- **Apply Tags** — Automatically tag matching notifications

### Creating a Rule

1. Go to **Settings** → **Rules**
2. Click **New Rule**
3. Choose rule type (query-based or view-linked)
4. Define the query or select a view
5. Configure rule name and description
6. Select actions to apply
7. Optionally add tags
8. Enable the rule and create it

### Example Rules

**Auto-archive Dependabot:**
```
Type: Query-based
Query: author:dependabot
Actions: Archive
```

**Auto-tag security alerts:**
```
Type: Query-based
Query: type:RepositoryVulnerabilityAlert
Actions: (none)
Tags: security
```

**Auto-star review requests:**
```
Type: Query-based
Query: reason:review_requested
Actions: Star
```

**Rule order matters:** Rules are processed from top to bottom. Put more specific rules before general ones.

**Learn more:** See the [Views and Rules Guide](features/views-and-rules.md#rules) for detailed information and best practices.

---

## 5. Keyboard Shortcuts

Octobud is designed for keyboard-first navigation. Most operations can be done without touching your mouse.

### Quick Reference

**Navigation:**
- `j` / `k` — Navigate notification list
- `Space` — Toggle notification detail
- `gg` — Jump to first notification
- `Shift + g` — Jump to last notification
- `[` / `]` — Navigate pages
- `Shift + j` / `Shift + k` — Navigate views
- `/` — Focus query input

**Actions:**
- `s` — Star (or selected items)
- `r` — Toggle read status
- `e` — Archive
- `z` — Open snooze dropdown
- `m` — Mute
- `t` — Open tag dropdown
- `i` — Allow back into inbox
- `o` — Open in GitHub
- `h` — Toggle shortcuts help

**Multiselect mode:**
- `v` — Toggle multiselect mode
- `x` — Toggle selection of focused item
- `a` — Cycle select all (page → all → none)

**Command Palette:**
- `Cmd + k` — Open command palette (prepopulated with view switcher)
- `Shift + Cmd + k` — Open empty command palette
- `Shift + v` — Open bulk actions palette

### Context-Aware Behavior

Many shortcuts automatically adapt to context:

- **In multiselect mode:** Actions apply to all selected items
- **Without selection:** Actions apply to the focused notification
- **In split view:** Actions check for multiselect selections first
- **In single detail view:** Only focused notification actions work

### Getting Help

Press `h` at any time to see all available keyboard shortcuts in a modal.

**Learn more:** See the [Keyboard Shortcuts Guide](features/keyboard-shortcuts.md) for complete documentation and conditional behaviors.

---

## Next Steps

- [Query Syntax Guide](features/query-syntax.md) — Learn the full query language
- [Views and Rules Guide](features/views-and-rules.md) — Master organization and automation
- [Keyboard Shortcuts Guide](features/keyboard-shortcuts.md) — Become a power user

