# Action Hints

Action hints provide the frontend with information about which actions would cause a notification to be dismissed from the current query view. This enables the UI to optimistically update its state after a notification is actioned without having to wait for a fully refreshed list of notifications from the backend.

## Overview

Action hints are computed dynamically based on:
1. The current notification's state (archived, muted, snoozed, filtered, etc.)
2. The active query filter
3. The query evaluator that determines if a notification matches the query

The system tests each possible action by simulating it on a cloned notification and checking if the modified notification would still match the current query. If the notification would no longer match after the action, that action is included in the `dismissedOn` list.

## Architecture

### Core Components

#### `ActionHints` Type
```go
type ActionHints struct {
    DismissedOn []string `json:"dismissedOn"`
}
```

The `DismissedOn` array contains action names (e.g., `"archive"`, `"mute"`, `"snooze"`) that would cause the notification to be removed from the current query view.

#### Computation Functions

**`ComputeActionHints`** (`backend/internal/query/hints.go`)
- Entry point that takes a notification, repository, and query string
- Creates a query evaluator
- Delegates to `ComputeActionHintsWithEvaluator` for efficiency

**`ComputeActionHintsWithEvaluator`** (`backend/internal/query/hints.go`)
- More efficient version that accepts a pre-created evaluator
- Used when computing hints for multiple notifications with the same query
- Tests all possible dismissive actions

**`wouldDismissOnAction`** (`backend/internal/query/hints.go`)
- Core logic that simulates an action on a cloned notification
- Checks if the modified notification would still match the query
- Returns `true` if the action would dismiss (notification no longer matches)

### Supported Actions

The system tests the following actions:

1. **Archive/Unarchive**
   - `archive`: Sets `Archived = true`
   - `unarchive`: Sets `Archived = false`

2. **Mute/Unmute**
   - `mute`: Sets `Muted = true`
   - `unmute`: Sets `Muted = false`

3. **Snooze/Unsnooze**
   - `snooze`: Sets `SnoozedUntil` to 24 hours in the future (only tested if not currently snoozed)
   - `unsnooze`: Clears `SnoozedUntil` (only tested if currently snoozed)

4. **Filter/Unfilter**
   - `filter`: Sets `Filtered = true` (only tested if not currently filtered)
   - `unfilter`: Sets `Filtered = false` (only tested if currently filtered)

### Actions That Never Dismiss

The following actions are explicitly excluded from dismissal hints (per UX decisions):

- **Read/Unread**: Never dismiss (Gmail-style UX - only refresh dismisses)
- **Star/Unstar**: Never dismiss (per UX decision)

### Integration Points

#### Backend API Response
Action hints are computed in `BuildResponse` (`backend/internal/core/notification/response.go`):
- Hints are computed for every notification in the response
- Uses a pre-created evaluator for efficiency when processing multiple notifications
- If query parsing fails, returns conservative hints (empty `dismissedOn` array)

#### Frontend Usage
The frontend uses action hints in `notificationActionController.ts`:
- Checks if an action is in the `dismissedOn` list before performing it
- Uses this information to determine if the notification should be removed from the view after the action
- Handles optimistic updates and view refresh based on dismissal status

## How It Works

### Step-by-Step Process

1. **Query Evaluation Setup**
   - The query string is parsed into an evaluator
   - The evaluator can determine if a notification matches the query

2. **Action Testing**
   For each possible action:
   - Clone the notification
   - Apply the action to the clone (modify the relevant field)
   - Check if the modified notification still matches the query using `evaluator.Matches()`
   - If it no longer matches, add the action to `dismissedOn`

3. **State-Aware Testing**
   - Only tests actions that make sense given the current state
   - For example, only tests `snooze` if the notification is not currently snoozed
   - Only tests `unsnooze` if the notification is currently snoozed

### Example

Given a notification that is:
- Not archived
- Not muted
- Matches query: `in:inbox`

The system would:
1. Test `archive`: Clone notification, set `Archived = true`, check if it matches `in:inbox`
   - Since `in:inbox` excludes archived items, this would return `false` (no longer matches)
   - Result: `archive` is added to `dismissedOn`

2. Test `mute`: Clone notification, set `Muted = true`, check if it matches `in:inbox`
   - Since `in:inbox` excludes muted items, this would return `false` (no longer matches)
   - Result: `mute` is added to `dismissedOn`

3. Test `unarchive`: Skip (notification is not archived)

4. Test `unmute`: Skip (notification is not muted)

Final result: `ActionHints{DismissedOn: ["archive", "mute"]}`

## Performance Considerations

### Efficient Evaluation
- Uses `ComputeActionHintsWithEvaluator` to reuse a single evaluator for multiple notifications
- Query parsing happens once per request, not per notification
- Action testing is fast (in-memory operations on cloned structs)

### Conservative Error Handling
- If query parsing fails, returns empty `dismissedOn` array (safe default)
- Never crashes or returns errors that would break the UI
- Gracefully handles missing repository data

## Query Semantics

Action hints respect the full query semantics, including:
- `in:` operators (inbox, archive, snoozed, filtered, anywhere)
- `is:` operators (read, unread, archived, muted, etc.)
- Field filters (repo, author, type, state, etc.)
- Logical operators (AND, OR, NOT)
- Parentheses grouping

This means action hints accurately reflect what would happen based on the actual query being executed, not just simple state checks.

## Testing

Tests exist in `backend/internal/query/hints_test.go` covering:
- Basic action dismissal scenarios
- Query-specific behavior
- State-aware action testing
- Edge cases and error handling

