# Sync Architecture

The sync system is responsible for periodically fetching notifications from GitHub's API and processing them into the local database. It uses a job queue (River) to handle the work asynchronously and maintain sync state to avoid duplicate processing.

## Overview

The sync process runs continuously in the background worker, periodically fetching new notifications from GitHub and processing them through a multi-stage pipeline:

1. **Periodic Sync Job**: Triggered on a configurable interval (default: 30 seconds)
2. **Fetch Notifications**: Retrieves new/updated notifications from GitHub API
3. **Queue Processing Jobs**: Creates individual jobs for each notification
4. **Process Notifications**: Upserts repositories, fetches subject data, and stores notifications
5. **Apply Rules**: Automatically applies matching rules to new notifications
6. **Update Sync State**: Tracks the latest notification timestamp for incremental syncs

## Architecture

### Components

#### Sync Service (`backend/internal/github/sync.go`)

The `SyncService` coordinates the entire sync process:

```go
type SyncService struct {
    client             Client
    clock              func() time.Time
    logger             *zap.Logger
    syncService        *sync.Service
    repositoryService  *repository.Service
    pullRequestService *pullrequest.Service
    queries            db.Store
}
```

**Key Methods:**

- **`FetchNotificationsToSync`**: Determines which notifications need to be synced
  - Retrieves sync state from database
  - Uses `LatestNotificationAt` or `LastSuccessfulPoll` as the `since` parameter
  - Fetches notifications from GitHub API
  - Updates sync state even when no notifications are found

- **`ProcessNotification`**: Processes a single notification thread
  - Upserts the repository
  - Fetches subject details (PR/Issue data) from GitHub
  - Extracts metadata (author, state, number, etc.)
  - Upserts pull request data if applicable
  - Upserts the notification with all metadata

- **`UpdateSyncStateAfterProcessing`**: Updates sync state after batch processing
  - Records the latest notification timestamp
  - Updates `LastSuccessfulPoll` timestamp

- **`RefreshSubjectData`**: Manually refreshes subject data for a notification
  - Used when subject data needs to be updated (e.g., PR state changed)
  - Fetches fresh data from GitHub and updates the notification

#### Sync State Service (`backend/internal/core/sync/sync.go`)

Manages sync state persistence:

```go
type SyncState struct {
    LastSuccessfulPoll   sql.NullTime
    LatestNotificationAt sql.NullTime
    UpdatedAt            time.Time
}
```

**Key Methods:**

- **`GetSyncState`**: Retrieves current sync state from database
- **`UpsertSyncState`**: Updates or creates sync state record

#### Job Workers

**SyncNotificationsWorker** (`backend/internal/jobs/sync_notifications.go`)
- Runs periodically (configurable interval, default 30 seconds)
- Fetches notifications from GitHub via `SyncService.FetchNotificationsToSync`
- Queues individual `ProcessNotification` jobs for each notification
- Updates sync state after queuing all jobs
- Uses unique job constraints to prevent duplicate sync jobs

**ProcessNotificationWorker** (`backend/internal/jobs/process_notification.go`)
- Processes individual notifications
- Calls `SyncService.ProcessNotification` to upsert data
- Applies matching rules to newly created notifications (not updates)
- Handles errors gracefully (rule application failures don't fail the job)

## Sync Flow

### Periodic Sync Cycle

```
┌─────────────────────────────────────────────────────────────┐
│  Periodic Sync Job (every 30s, configurable)                 │
│  SyncNotificationsWorker                                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  FetchNotificationsToSync                                    │
│  - Get sync state (LatestNotificationAt)                     │
│  - Fetch from GitHub API with 'since' parameter              │
│  - Return list of NotificationThread objects                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  Queue ProcessNotification Jobs                              │
│  - For each notification thread:                             │
│    * Marshal thread data                                     │
│    * Insert ProcessNotification job                          │
│  - Track latest update time                                  │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  UpdateSyncStateAfterProcessing                              │
│  - Update LatestNotificationAt                               │
│  - Update LastSuccessfulPoll                                 │
└─────────────────────────────────────────────────────────────┘
```

### Individual Notification Processing

```
┌─────────────────────────────────────────────────────────────┐
│  ProcessNotificationWorker                                   │
│  - Unmarshal notification thread data                        │
│  - Check if notification already exists                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  ProcessNotification                                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ 1. Upsert Repository                                  │   │
│  │    - Create or update repository record              │   │
│  │    - Store full repository metadata                  │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ 2. Fetch Subject Data                                │   │
│  │    - Fetch PR/Issue details from GitHub API          │   │
│  │    - Extract metadata (author, state, number, etc.)  │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ 3. Upsert Pull Request (if applicable)               │   │
│  │    - Extract PR data from subject JSON               │   │
│  │    - Store in pull_requests table                    │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ 4. Upsert Notification                                │   │
│  │    - Store notification with all metadata            │   │
│  │    - Link to repository and pull request             │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  Apply Rules (only for new notifications)                     │
│  - Match notification against all active rules              │
│  - Apply matching rule actions (archive, mute, filter, etc.) │
└─────────────────────────────────────────────────────────────┘
```

## Sync State Management

### Incremental Sync

The sync system uses incremental fetching to avoid processing the same notifications repeatedly:

1. **Initial Sync**: When no sync state exists, fetches all notifications
2. **Incremental Sync**: Uses `LatestNotificationAt` as the `since` parameter
3. **Fallback**: If `LatestNotificationAt` is not set, uses `LastSuccessfulPoll`

### State Updates

Sync state is updated in two scenarios:

1. **After Fetching**: Even when no notifications are found, `LastSuccessfulPoll` is updated
2. **After Processing**: `LatestNotificationAt` is set to the most recent notification's `UpdatedAt` timestamp

This ensures the system always knows when it last successfully synced, even if no new notifications were found.

## Error Handling

### Graceful Degradation

- **Subject Fetch Failures**: Logged as warnings but don't fail the job (subject data is optional)
- **Pull Request Metadata Failures**: Logged as warnings but don't fail the job (PR metadata is optional)
- **Rule Application Failures**: Logged but don't fail the job (rules are best-effort)
- **Sync State Update Failures**: Logged but don't fail job queuing (jobs are already queued)

### Retry Logic

River handles job retries automatically:
- Failed jobs are retried with exponential backoff
- Maximum retry attempts are configurable
- Permanent failures are marked as such

## Configuration

### Worker Setup (`backend/cmd/worker/main.go`)

The worker is configured with:

- **Sync Interval**: Configurable via `SYNC_INTERVAL` environment variable (default: 20 seconds)
- **Queue Configuration**:
  - `sync_notifications`: 1 worker (prevents concurrent syncs)
  - `process_notification`: 10 workers (allows parallel processing)
  - `apply_rule`: 10 workers (allows parallel rule application)

- **Unique Job Constraints**: Prevents duplicate sync jobs from running simultaneously
  - Checks for jobs in states: Available, Pending, Running, Retryable, Scheduled

### Periodic Job

The sync job is registered as a periodic job with:
- Configurable interval (default: 30 seconds)
- `RunOnStart: true` - runs immediately when worker starts
- Unique constraints to prevent duplicate jobs

## Data Flow

### Repository Upsert

When processing a notification:
1. Repository data is extracted from the notification thread
2. Repository is upserted (create or update) with full metadata
3. Repository ID is used to link the notification

### Subject Data Fetching

For each notification:
1. Subject URL is extracted from the notification thread
2. Subject data is fetched from GitHub API (PR/Issue details)
3. Metadata is extracted:
   - Author login and ID
   - Subject number (PR/Issue number)
   - Subject state (open, closed, merged)
   - Subject merged status
   - Subject state reason

### Pull Request Processing

If the subject is a PullRequest:
1. PR data is extracted from subject JSON
2. PR is upserted to `pull_requests` table
3. PR ID is linked to the notification

### Notification Upsert

The notification is stored with:
- GitHub notification metadata (ID, reason, unread status, etc.)
- Repository link
- Pull request link (if applicable)
- Subject metadata (type, title, URL, etc.)
- Raw payloads for future reference

## Rule Application

After a notification is processed:
1. System checks if this is a new notification (not an update)
2. If new, matches the notification against all active rules
3. Applies actions from matching rules (archive, mute, filter, assign tags, etc.)

This happens asynchronously in a separate job to avoid blocking notification processing.

## Performance Considerations

### Parallel Processing

- Multiple notifications can be processed in parallel (10 workers)
- Repository upserts are idempotent (safe to run concurrently)
- Rule application is parallelized (10 workers)

### Efficient Fetching

- Uses incremental sync to only fetch new/updated notifications
- GitHub API pagination is handled automatically
- Sync state prevents duplicate processing

### Database Efficiency

- Uses upserts to avoid duplicate key errors
- Batch operations where possible
- Indexes on frequently queried fields (github_id, repository_id, etc.)

## Monitoring and Observability

### Logging

The sync system logs:
- Sync job start/completion
- Number of notifications queued
- Errors during fetching and processing
- Warnings for optional operation failures

### Metrics

Key metrics to monitor:
- Sync job frequency and duration
- Number of notifications processed per sync
- Error rates for different operations
- Rule application success rates

