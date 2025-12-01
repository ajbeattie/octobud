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
- **Starred** — All starred notifications (not muted)
- **Snoozed** - All currently snoozed notifications (not muted)
- **Archived** — All archived notifications (not muted)
- **Everything** — All notifications, including muted ones

### Creating a View

1. Type a query in the search bar (e.g., `type:pullrequest reason:review_requested`)
2. Click **Save** or use the **+** button in the sidebar
3. Give it a name, choose an icon, and save

### Query Basics

Queries filter notifications using a simple syntax. Combine multiple filters:

```
type:pullrequest reason:review_requested is:unread
(reason:mention OR reason:review_requested) AND is:unread
```

**Operators:** Space = AND, `OR` = OR, `NOT` or `-` = NOT, `()` = grouping

**Common filters:** `is:unread`, `type:PullRequest`, `reason:review_requested`, `repo:owner/name`, `org:name`, `tags:name`

**Learn more:** See the [Query Syntax Guide](features/query-syntax.md) for complete documentation.

---

## 3. Tag Basics

Tags are custom labels you can assign to notifications to organize them beyond the built-in states.

### Creating and Using Tags

1. Go to **Settings** → **Tags** to create tags
2. Press `t` on any notification to add/remove tags
3. Use `tags:name` in queries to filter by tag

Tags can be assigned manually or automatically via rules.

**Learn more:** Tags are covered in the [Views and Rules Guide](features/views-and-rules.md#tags).

---

## 4. Rule Basics

Rules automatically apply actions to notifications that match a query. They help automate your workflow by handling routine tasks.

### How Rules Work

Rules automatically apply actions to notifications matching a query. They run in order when new notifications arrive.

**Rule types:**
- **Query-based** — Independent query (e.g., auto-archive `author:dependabot`)
- **View-linked** — Uses a view's query and stays in sync with view changes

**Available actions:** Skip Inbox, Mark as Read, Archive, Star, Mute, Apply Tags

**Creating a rule:**
1. Go to **Settings** → **Rules** → **New Rule**
2. Choose type, define query/view, select actions, optionally add tags
3. Enable and save

**Rule order matters** — More specific rules should come before general ones.

**Learn more:** See the [Views and Rules Guide](features/views-and-rules.md#rules) for detailed information and best practices.

---

## 5. Keyboard Shortcuts

Octobud is designed for keyboard-first navigation. Press `h` to see all shortcuts.

**Essentials:**
- `j` / `k` — Navigate list
- `s`, `r`, `e`, `z`, `m`, `t` — Actions (star, read, archive, snooze, mute, tag)
- `v` — Toggle multiselect mode
- `Cmd + k` — Command palette

**Learn more:** See the [Keyboard Shortcuts Guide](features/keyboard-shortcuts.md) for complete documentation.

---

## Next Steps

- [Query Syntax Guide](features/query-syntax.md) — Learn the full query language
- [Views and Rules Guide](features/views-and-rules.md) — Master organization and automation
- [Keyboard Shortcuts Guide](features/keyboard-shortcuts.md) — Become a power user

