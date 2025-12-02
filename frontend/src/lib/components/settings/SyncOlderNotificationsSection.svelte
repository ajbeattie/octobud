<!-- Copyright (C) 2025 Austin Beattie

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>. -->

<script lang="ts">
	import { onMount } from "svelte";
	import { getSyncState, syncOlderNotifications, type SyncState } from "$lib/api/user";
	import { toastStore } from "$lib/stores/toastStore";
	import ConfirmDialog from "$lib/components/dialogs/ConfirmDialog.svelte";

	type DaySelection = number | "custom";

	let syncState: SyncState | null = null;
	let isLoading = true;
	let isSubmitting = false;
	let error = "";
	let showConfirmDialog = false;

	// Form state
	let selectedDays: DaySelection = 30;
	let customDays: number | null = null;
	let maxNotifications: number | null = null;
	let syncUnreadOnly = false;

	// Predefined day options
	const dayOptions = [30, 60, 90];

	// Validation limits (must match backend)
	const MAX_SYNC_DAYS = 3650; // 10 years
	const MAX_NOTIFICATION_COUNT = 100000;

	// Show warning for large sync periods
	$: showLargeSyncWarning =
		(selectedDays === "custom" && customDays !== null && customDays > 90) ||
		(typeof selectedDays === "number" && selectedDays > 90);

	// Get effective days for submission
	function getEffectiveDays(): number {
		if (selectedDays === "custom") return customDays || 30;
		return selectedDays;
	}

	// Format date for display
	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString(undefined, {
			year: "numeric",
			month: "short",
			day: "numeric",
			hour: "2-digit",
			minute: "2-digit",
		});
	}

	onMount(async () => {
		try {
			syncState = await getSyncState();
		} catch (err) {
			console.error("Failed to fetch sync state:", err);
			error = "Failed to load sync state";
		} finally {
			isLoading = false;
		}
	});

	function handleDaysChange(option: DaySelection) {
		selectedDays = option;
		if (option === "custom") {
			customDays = customDays || 30;
		}
	}

	function handleSubmit() {
		error = "";

		const days = getEffectiveDays();

		// Validate before showing dialog
		if (days < 1) {
			error = "Please enter a valid number of days";
			return;
		}
		if (days > MAX_SYNC_DAYS) {
			error = `Days cannot exceed ${MAX_SYNC_DAYS}`;
			return;
		}

		if (maxNotifications !== null) {
			if (maxNotifications < 1) {
				error = "Maximum notifications must be at least 1";
				return;
			}
			if (maxNotifications > MAX_NOTIFICATION_COUNT) {
				error = `Maximum notifications cannot exceed ${MAX_NOTIFICATION_COUNT}`;
				return;
			}
		}

		// Show confirmation dialog
		showConfirmDialog = true;
	}

	async function handleConfirmedSync() {
		showConfirmDialog = false;
		isSubmitting = true;

		try {
			const days = getEffectiveDays();

			await syncOlderNotifications({
				days,
				maxCount: maxNotifications,
				unreadOnly: syncUnreadOnly,
			});

			toastStore.success(`Syncing ${days} more days of notifications...`);

			// Refresh sync state to show updated oldest timestamp
			syncState = await getSyncState();
		} catch (err) {
			console.error("Failed to sync older notifications:", err);
			error = err instanceof Error ? err.message : "Failed to start sync";
			toastStore.error(error);
		} finally {
			isSubmitting = false;
		}
	}

	function handleCancelSync() {
		showConfirmDialog = false;
	}

	$: canSync = syncState?.oldestNotificationSyncedAt != null;

	// Compute effective days reactively (must reference variables directly for Svelte reactivity)
	$: effectiveDays = selectedDays === "custom" ? customDays || 30 : selectedDays;
	$: confirmDialogBody = `This will fetch notifications from ${effectiveDays} days before your oldest synced notification. This may take a while depending on how many notifications exist.`;
</script>

<div class="space-y-4">
	<div>
		<h3 class="text-md font-medium text-gray-900 dark:text-gray-100">Sync Past Notifications</h3>
		<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
			Fetch older notifications that weren't included in your initial sync.
		</p>
	</div>

	{#if isLoading}
		<div class="flex items-center justify-center py-8 text-sm text-gray-500 dark:text-gray-400">
			Loading...
		</div>
	{:else if !canSync}
		<div
			class="rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600 dark:border-gray-700 dark:bg-gray-900/50 dark:text-gray-400"
		>
			Complete initial setup first. Once notifications have been synced, you can fetch older ones
			here.
		</div>
	{:else}
		<!-- Form card -->
		<div
			class="rounded-lg border border-gray-200 bg-gray-50 p-5 dark:border-gray-800 dark:bg-gray-900/60"
		>
			<!-- Current oldest notification info -->
			{#if syncState?.oldestNotificationSyncedAt}
				<div
					class="mb-5 rounded-lg border border-blue-200 bg-blue-50 p-3 text-xs dark:border-blue-700/50 dark:bg-blue-900/20"
				>
					<span class="text-blue-600 dark:text-blue-400">Oldest synced notification:</span>
					<span class="ml-1 font-medium text-blue-700 dark:text-blue-300">
						{formatDate(syncState.oldestNotificationSyncedAt)}
					</span>
				</div>
			{/if}

			{#if error}
				<div
					class="mb-5 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400"
					role="alert"
				>
					{error}
				</div>
			{/if}

			<form on:submit|preventDefault={handleSubmit} class="space-y-6">
				<!-- Days selector -->
				<div>
					<p class="text-sm font-medium text-gray-700 dark:text-gray-300">
						Sync more past notifications
					</p>
					<div class="mt-3 flex flex-wrap gap-2">
						{#each dayOptions as days (days)}
							<button
								type="button"
								on:click={() => handleDaysChange(days)}
								disabled={isSubmitting}
								class="rounded-lg border px-3 py-1.5 text-sm transition-colors cursor-pointer {selectedDays ===
								days
									? 'border-violet-500 bg-violet-50 text-violet-700 dark:border-violet-400 dark:bg-violet-500/10 dark:text-violet-400'
									: 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:bg-gray-700'}"
							>
								{days} days
							</button>
						{/each}
						<button
							type="button"
							on:click={() => handleDaysChange("custom")}
							disabled={isSubmitting}
							class="rounded-lg border px-3 py-1.5 text-sm transition-colors cursor-pointer {selectedDays ===
							'custom'
								? 'border-violet-500 bg-violet-50 text-violet-700 dark:border-violet-400 dark:bg-violet-500/10 dark:text-violet-400'
								: 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:bg-gray-700'}"
						>
							Custom
						</button>
					</div>

					{#if selectedDays === "custom"}
						<input
							type="number"
							min="1"
							max={MAX_SYNC_DAYS}
							placeholder="Enter number of days"
							bind:value={customDays}
							disabled={isSubmitting}
							class="mt-3 w-32 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-900 placeholder-gray-400 focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500 dark:focus:border-violet-400 dark:focus:ring-violet-400"
						/>
					{/if}

					{#if showLargeSyncWarning}
						<p class="mt-3 text-xs text-amber-600 dark:text-amber-500">
							Large sync periods may take a while and include many notifications.
						</p>
					{/if}
				</div>

				<!-- Advanced options -->
				<details class="group">
					<summary
						class="flex cursor-pointer list-none items-center gap-1 text-xs font-medium text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
					>
						<svg
							class="h-3 w-3 transition-transform group-open:rotate-90"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M9 5l7 7-7 7"
							/>
						</svg>
						Advanced options
					</summary>

					<div class="mt-4 space-y-4 pl-4">
						<!-- Max notifications -->
						<div>
							<label
								for="maxNotifications"
								class="block text-xs font-medium text-gray-700 dark:text-gray-300"
							>
								Maximum notifications
							</label>
							<input
								id="maxNotifications"
								type="number"
								min="1"
								max={MAX_NOTIFICATION_COUNT}
								placeholder="No limit"
								bind:value={maxNotifications}
								disabled={isSubmitting}
								class="mt-2 w-48 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-900 placeholder-gray-400 focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500 dark:focus:border-violet-400 dark:focus:ring-violet-400"
							/>
						</div>

						<!-- Unread only -->
						<label class="flex cursor-pointer items-center gap-2">
							<input
								type="checkbox"
								bind:checked={syncUnreadOnly}
								disabled={isSubmitting}
								class="h-4 w-4 rounded border-gray-300 text-violet-600 focus:ring-violet-500 dark:border-gray-600 dark:bg-gray-700"
							/>
							<span class="text-xs text-gray-700 dark:text-gray-300">Only unread notifications</span
							>
						</label>
					</div>
				</details>

				<!-- Submit button -->
				<div class="flex justify-end">
					<button
						type="submit"
						disabled={isSubmitting}
						class="inline-flex items-center gap-2 rounded-full bg-indigo-600 px-4 py-2 text-xs font-semibold text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
					>
						{#if isSubmitting}
							<svg class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
								<circle
									class="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									stroke-width="4"
								></circle>
								<path
									class="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								></path>
							</svg>
							Syncing...
						{:else}
							Sync older notifications
						{/if}
					</button>
				</div>
			</form>
		</div>
	{/if}
</div>

<ConfirmDialog
	open={showConfirmDialog}
	title="Sync older notifications?"
	body={confirmDialogBody}
	confirmLabel="Start sync"
	cancelLabel="Cancel"
	confirming={isSubmitting}
	onConfirm={handleConfirmedSync}
	onCancel={handleCancelSync}
/>
