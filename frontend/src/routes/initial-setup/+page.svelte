<script lang="ts">
	// Copyright (C) 2025 Austin Beattie
	//
	// This program is free software: you can redistribute it and/or modify
	// it under the terms of the GNU Affero General Public License as
	// published by the Free Software Foundation, either version 3 of the
	// License, or (at your option) any later version.
	//
	// This program is distributed in the hope that it will be useful,
	// but WITHOUT ANY WARRANTY; without even the implied warranty of
	// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	// GNU Affero General Public License for more details.
	//
	// You should have received a copy of the GNU Affero General Public License
	// along with this program.  If not, see <https://www.gnu.org/licenses/>.

	import { onMount } from "svelte";
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import {
		isAuthenticated,
		hasSyncSettingsConfigured,
		fetchUserInfo,
		syncSettings,
		logout,
	} from "$lib/stores/authStore";
	import { updateSyncSettings } from "$lib/api/user";
	import { get } from "svelte/store";
	import { resetSWPollState } from "$lib/utils/serviceWorkerRegistration";

	type DaySelection = number | "custom" | "all";
	let selectedOption: DaySelection = 30; // Default to 30 days
	let customDays: number | null = null;
	let maxNotifications: number | null = null;
	let syncUnreadOnly = false;
	let error = "";
	let isLoading = false;
	let isSubmitting = false;

	// Predefined day options
	const dayOptions = [30, 60, 90];

	// Validation limits (must match backend)
	const MAX_SYNC_DAYS = 3650; // 10 years
	const MAX_NOTIFICATION_COUNT = 100000;

	// Show warning for large sync periods
	$: showLargeSyncWarning =
		selectedOption === "all" ||
		(selectedOption === "custom" && customDays !== null && customDays > 90);

	// Get effective days for submission
	function getEffectiveDays(): number | null {
		if (selectedOption === "all") return null;
		if (selectedOption === "custom") return customDays;
		return selectedOption;
	}

	onMount(async () => {
		// Check if already authenticated and has sync settings configured
		if (!get(isAuthenticated)) {
			await goto(resolve("/login"));
			return;
		}

		// Fetch user info to check sync settings
		try {
			await fetchUserInfo();
			const hasConfigured = hasSyncSettingsConfigured();
			if (hasConfigured) {
				// Already configured, redirect to main app
				await goto(resolve("/views/inbox"));
			}
		} catch (err) {
			console.error("Failed to fetch user info:", err);
			// Continue to setup page even if fetch fails
		}
	});

	function handleDaysChange(option: DaySelection) {
		selectedOption = option;
		if (option === "custom") {
			customDays = customDays || 30;
		} else if (option !== "all") {
			customDays = null;
		}
	}

	async function handleSubmit() {
		error = "";
		isSubmitting = true;

		try {
			const initialSyncDays = getEffectiveDays();

			// Validate - allow null (all time) but reject invalid custom values
			if (selectedOption === "custom") {
				if (!customDays || customDays < 1) {
					error = "Please enter a valid number of days";
					isSubmitting = false;
					return;
				}
				if (customDays > MAX_SYNC_DAYS) {
					error = `Days cannot exceed ${MAX_SYNC_DAYS} (10 years)`;
					isSubmitting = false;
					return;
				}
			}

			// Validate max notifications if provided
			if (maxNotifications !== null && maxNotifications > MAX_NOTIFICATION_COUNT) {
				error = `Maximum notifications cannot exceed ${MAX_NOTIFICATION_COUNT.toLocaleString()}`;
				isSubmitting = false;
				return;
			}

			await updateSyncSettings({
				initialSyncDays,
				initialSyncMaxCount: maxNotifications || null,
				initialSyncUnreadOnly: syncUnreadOnly,
				setupCompleted: true,
			});

			// Refresh user info to update store
			await fetchUserInfo();

			// Reset SW poll state so all notifications are treated as new
			await resetSWPollState();

			// Set sync start timestamp so the banner shows until notifications arrive
			if (typeof window !== "undefined") {
				localStorage.setItem("octobud:sync-started-at", Date.now().toString());
			}

			// Redirect to main app
			await goto(resolve("/views/inbox"));
		} catch (err) {
			error = err instanceof Error ? err.message : "Failed to save sync settings";
			isSubmitting = false;
		}
	}

	// Use default recommendation: 30 days
	function useRecommended() {
		selectedOption = 30;
		customDays = null;
		maxNotifications = null;
		syncUnreadOnly = false;
	}

	async function handleLogout() {
		await logout();
		await goto(resolve("/login"));
	}
</script>

<div class="relative min-h-screen overflow-y-auto bg-white pt-16 dark:bg-gray-950">
	<!-- Back to login button -->
	<button
		type="button"
		on:click={handleLogout}
		class="absolute right-6 top-6 inline-flex items-center gap-1 text-sm text-gray-400 transition-colors hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 sm:left-10 sm:top-6 cursor-pointer"
	>
		<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M10 19l-7-7m0 0l7-7m-7 7h18"
			/>
		</svg>
		Back to login
	</button>

	<div class="mx-auto w-full max-w-xl px-4 py-10 sm:px-6 sm:py-12">
		<!-- Header -->
		<div class="flex items-center gap-4">
			<div
				class="flex h-16 w-16 flex-shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-600"
			>
				<img src="/baby_octo.svg" alt="Octobud" class="h-9 w-9" />
			</div>
			<div>
				<h1 class="text-xl font-semibold text-gray-900 dark:text-white">Welcome to Octobud!</h1>
				<p class="text-sm text-gray-500 dark:text-gray-400">
					Configure the initial sync of your GitHub notifications to get started
				</p>
				<a
					href="https://github.com/ajbeattie/octobud/blob/main/docs/start-here.md#initial-sync"
					target="_blank"
					rel="noopener noreferrer"
					class="mt-1 inline-flex items-center gap-1 text-xs text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300"
				>
					Learn more about initial sync
					<svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
						/>
					</svg>
				</a>
			</div>
		</div>

		<!-- Main form card -->
		<form class="mt-12" on:submit|preventDefault={handleSubmit}>
			{#if error}
				<div
					class="mb-6 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400"
					role="alert"
				>
					{error}
				</div>
			{/if}

			<div
				class="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900 sm:p-8"
			>
				<!-- Time period selection -->
				<div>
					<h3 class="text-sm font-semibold text-gray-900 dark:text-white">
						Sync notifications from
					</h3>
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
						Choose a time period or sync everything
					</p>
					<div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
						{#each dayOptions as days (days)}
							<button
								type="button"
								on:click={() => handleDaysChange(days)}
								class="relative rounded-xl border-2 px-4 py-3 text-center transition-all cursor-pointer {selectedOption ===
								days
									? 'border-violet-500 bg-violet-50 ring-1 ring-violet-500 dark:border-violet-400 dark:bg-violet-500/10 dark:ring-violet-400'
									: 'border-gray-200 bg-gray-50 hover:border-gray-300 hover:bg-gray-100 dark:border-gray-700 dark:bg-gray-800/50 dark:hover:border-gray-600 dark:hover:bg-gray-700/50'}"
								disabled={isSubmitting}
							>
								<span
									class="block text-lg font-semibold {selectedOption === days
										? 'text-violet-600 dark:text-violet-400'
										: 'text-gray-900 dark:text-white'}"
								>
									{days}
								</span>
								<span
									class="block text-xs {selectedOption === days
										? 'text-violet-500 dark:text-violet-400'
										: 'text-gray-500 dark:text-gray-400'}"
								>
									days
								</span>
							</button>
						{/each}
						<button
							type="button"
							on:click={() => handleDaysChange("all")}
							class="relative rounded-xl border-2 px-4 py-3 text-center transition-all cursor-pointer {selectedOption ===
							'all'
								? 'border-violet-500 bg-violet-50 ring-1 ring-violet-500 dark:border-violet-400 dark:bg-violet-500/10 dark:ring-violet-400'
								: 'border-gray-200 bg-gray-50 hover:border-gray-300 hover:bg-gray-100 dark:border-gray-700 dark:bg-gray-800/50 dark:hover:border-gray-600 dark:hover:bg-gray-700/50'}"
							disabled={isSubmitting}
						>
							<span
								class="block text-lg font-semibold {selectedOption === 'all'
									? 'text-violet-600 dark:text-violet-400'
									: 'text-gray-900 dark:text-white'}"
							>
								All
							</span>
							<span
								class="block text-xs {selectedOption === 'all'
									? 'text-violet-500 dark:text-violet-400'
									: 'text-gray-500 dark:text-gray-400'}"
							>
								time
							</span>
						</button>
					</div>

					<!-- Custom input -->
					<button
						type="button"
						on:click={() => handleDaysChange("custom")}
						class="mt-3 w-full rounded-xl border-2 px-4 py-3 text-left transition-all cursor-pointer {selectedOption ===
						'custom'
							? 'border-violet-500 bg-violet-50 ring-1 ring-violet-500 dark:border-violet-400 dark:bg-violet-500/10 dark:ring-violet-400'
							: 'border-gray-200 bg-gray-50 hover:border-gray-300 hover:bg-gray-100 dark:border-gray-700 dark:bg-gray-800/50 dark:hover:border-gray-600 dark:hover:bg-gray-700/50'}"
						disabled={isSubmitting}
					>
						<span
							class="text-sm font-medium {selectedOption === 'custom'
								? 'text-violet-600 dark:text-violet-400'
								: 'text-gray-700 dark:text-gray-300'}"
						>
							Custom number of days
						</span>
					</button>

					{#if selectedOption === "custom"}
						<div class="mt-3">
							<input
								type="number"
								min="1"
								max={MAX_SYNC_DAYS}
								placeholder="Enter number of days"
								bind:value={customDays}
								disabled={isSubmitting}
								class="w-full rounded-xl border-2 border-gray-200 bg-white px-4 py-3 text-gray-900 placeholder-gray-400 transition-colors focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500 dark:focus:border-violet-400 dark:focus:ring-violet-400"
							/>
						</div>
					{/if}

					{#if showLargeSyncWarning}
						<div
							class="mt-4 flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-700/50 dark:bg-amber-900/20"
						>
							<svg
								class="mt-0.5 h-5 w-5 flex-shrink-0 text-amber-500 dark:text-amber-400"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
								/>
							</svg>
							<div class="text-sm text-amber-800 dark:text-amber-300">
								{#if selectedOption === "all"}
									<span class="font-medium">This may take a while.</span> Syncing all notifications could
									include thousands of items.
								{:else}
									<span class="font-medium">Large sync.</span> Consider starting smaller-you can always
									sync more later.
								{/if}
							</div>
						</div>
					{/if}
				</div>

				<!-- Advanced options -->
				<details class="group mt-8 border-t border-gray-200 pt-6 dark:border-gray-700">
					<summary
						class="flex cursor-pointer list-none items-center justify-between text-sm font-medium text-gray-700 dark:text-gray-300"
					>
						<span>Advanced options</span>
						<svg
							class="h-5 w-5 text-gray-400 transition-transform group-open:rotate-180"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M19 9l-7 7-7-7"
							/>
						</svg>
					</summary>

					<div class="mt-4 space-y-4">
						<!-- Max notifications -->
						<div>
							<label
								for="maxNotifications"
								class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>
								Maximum notifications
							</label>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
								Stop syncing after reaching this limit
							</p>
							<input
								id="maxNotifications"
								type="number"
								min="1"
								max={MAX_NOTIFICATION_COUNT}
								placeholder="No limit"
								bind:value={maxNotifications}
								disabled={isSubmitting}
								class="mt-2 w-full rounded-xl border-2 border-gray-200 bg-white px-4 py-2.5 text-gray-900 placeholder-gray-400 transition-colors focus:border-violet-500 focus:outline-none focus:ring-1 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500 dark:focus:border-violet-400 dark:focus:ring-violet-400"
							/>
						</div>

						<!-- Unread only -->
						<label class="flex cursor-pointer items-start gap-3">
							<input
								type="checkbox"
								bind:checked={syncUnreadOnly}
								disabled={isSubmitting}
								class="mt-1 h-4 w-4 rounded border-gray-300 text-violet-600 focus:ring-violet-500 dark:border-gray-600 dark:bg-gray-700"
							/>
							<div>
								<span class="text-sm font-medium text-gray-700 dark:text-gray-300">
									Only unread notifications
								</span>
								<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
									Skip notifications you've already read on GitHub
								</p>
							</div>
						</label>
					</div>
				</details>
			</div>

			<!-- Actions -->
			<div class="mt-8 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<button
					type="button"
					on:click={useRecommended}
					disabled={isSubmitting}
					class="order-2 text-sm text-gray-500 transition-colors hover:text-gray-700 disabled:opacity-50 dark:text-gray-400 dark:hover:text-gray-300 sm:order-1 cursor-pointer"
				>
					Reset to recommended
				</button>
				<button
					type="submit"
					disabled={isSubmitting}
					class="order-1 inline-flex items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-indigo-600 to-violet-600 px-6 py-2.5 text-sm font-semibold text-white transition-all hover:from-indigo-500 hover:to-violet-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-gray-950 sm:order-2 cursor-pointer"
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
						Starting sync...
					{:else}
						Start syncing
						<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M13 7l5 5m0 0l-5 5m5-5H6"
							/>
						</svg>
					{/if}
				</button>
			</div>
		</form>

		<!-- Footer note -->
		<p class="mt-8 text-center text-xs text-gray-400 dark:text-gray-500">
			You can sync past notifications later from the settings page.
		</p>
	</div>
</div>
