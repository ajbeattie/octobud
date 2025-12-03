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
		login,
		isAuthenticated,
		hasSyncSettingsConfigured,
		fetchUserInfo,
	} from "$lib/stores/authStore";
	import { get } from "svelte/store";
	import { resetSWPollState, ensureSWPolling } from "$lib/utils/serviceWorkerRegistration";
	import { ApiUnreachableError } from "$lib/api/fetch";

	let username = "";
	let password = "";
	let error = "";
	let isApiUnreachable = false;
	let isLoading = false;

	onMount(async () => {
		// Redirect if already authenticated
		if (get(isAuthenticated)) {
			// Check if sync settings are configured
			try {
				await fetchUserInfo();
				if (hasSyncSettingsConfigured()) {
					await goto(resolve("/views/inbox"));
				} else {
					await goto(resolve("/initial-setup"));
				}
			} catch (err) {
				// If we can't fetch user info, redirect to setup to be safe
				await goto(resolve("/initial-setup"));
			}
		}
	});

	async function handleSubmit() {
		error = "";
		isApiUnreachable = false;
		isLoading = true;

		try {
			await login(username, password);

			// Ensure SW is ready and polling after login
			// This waits for the SW controller to be available and starts polling
			// resetSWPollState clears the poll state so all notifications are treated as new
			await ensureSWPolling();
			await resetSWPollState();

			// After login, check if sync settings are configured
			// Login already calls fetchUserInfo, so check directly
			if (hasSyncSettingsConfigured()) {
				await goto(resolve("/views/inbox"));
			} else {
				await goto(resolve("/initial-setup"));
			}
		} catch (err) {
			if (err instanceof ApiUnreachableError) {
				isApiUnreachable = true;
				error = "Unable to connect to the API server";
			} else {
				error = err instanceof Error ? err.message : "Login failed";
			}
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-950 px-4 py-12 sm:px-6 lg:px-8">
	<!-- Main card - horizontally laid out on large screens -->
	<div
		class="w-full max-w-5xl rounded-3xl border border-gray-200 bg-white shadow-2xl dark:border-gray-800 dark:bg-gray-900 lg:flex"
	>
		<!-- Left column: Logo and branding -->
		<div class="flex flex-col justify-between px-8 py-12 lg:w-2/5">
			<!-- Top section: Logo and text -->
			<div class="flex w-full items-start gap-6 pt-2">
				<!-- Logo subcolumn -->
				<div class="flex-shrink-0">
					<div
						class="flex h-20 w-20 items-center justify-center rounded-2xl bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-500 shadow-lg"
					>
						<img src="/baby_octo.svg" alt="Octobud icon" class="h-12 w-12" />
					</div>
				</div>
				<!-- Text subcolumn -->
				<div class="flex-1">
					<h1 class="text-2xl font-semibold text-gray-900 dark:text-gray-100">
						Sign in to Octobud
					</h1>
					<p class="text-sm text-gray-600 dark:text-gray-400">Default credentials:</p>
					<p class="mt-0.5">
						<span
							class="inline-block rounded-md bg-gray-100 px-2 py-0.5 font-mono text-[13px] text-gray-700 dark:bg-gray-800 dark:text-gray-300"
							>octobud / octobud</span
						>
					</p>
				</div>
			</div>

			<!-- Bottom section: Setup documentation link -->
			<div class="mt-auto">
				<a
					href="https://github.com/ajbeattie/octobud#quick-start"
					target="_blank"
					rel="noopener noreferrer"
					class="inline-flex items-center gap-1.5 text-sm text-indigo-600 transition-colors hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
				>
					<span>Setup documentation</span>
					<svg
						class="h-3.5 w-3.5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
						/>
					</svg>
				</a>
			</div>
		</div>

		<!-- Right column: Login form -->
		<div class="flex flex-col px-8 py-8 lg:w-3/5">
			<div
				class="rounded-2xl border border-gray-100 bg-gray-50 p-6 dark:border-gray-800 dark:bg-gray-800/50 sm:p-8"
			>
				<form class="flex h-full flex-col" on:submit|preventDefault={handleSubmit}>
					{#if error}
						<div
							class="mb-6 rounded-lg border {isApiUnreachable
								? 'border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-900/20'
								: 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-900/20'} p-4 text-sm"
							role="alert"
						>
							{#if isApiUnreachable}
								<div class="flex items-start gap-3">
									<svg
										class="h-5 w-5 flex-shrink-0 text-amber-600 dark:text-amber-400"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
										/>
									</svg>
									<div class="flex-1">
										<p class="font-semibold text-amber-800 dark:text-amber-200">
											Unable to connect to API
										</p>
										<p class="mt-1 text-amber-700 dark:text-amber-300">
											The backend server appears to be offline or unreachable. Please make sure all
											services are running.
										</p>
										<a
											href="https://github.com/ajbeattie/octobud#quick-start"
											target="_blank"
											rel="noopener noreferrer"
											class="mt-2 inline-flex items-center gap-1 text-amber-700 underline hover:text-amber-900 dark:text-amber-400 dark:hover:text-amber-300"
										>
											View setup documentation
											<svg
												class="h-3.5 w-3.5"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="2"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
												/>
											</svg>
										</a>
									</div>
								</div>
							{:else}
								<span class="text-red-800 dark:text-red-400">{error}</span>
							{/if}
						</div>
					{/if}

					<!-- Form fields -->
					<div class="flex-1 space-y-5">
						<div>
							<label
								for="username"
								class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>
								Username
							</label>
							<input
								id="username"
								name="username"
								type="text"
								required
								class="mt-2 block w-full rounded-lg border border-gray-200 bg-white px-4 py-3 text-base text-gray-900 placeholder-gray-500 transition-colors focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100 dark:placeholder-gray-500 dark:focus:border-indigo-400 dark:focus:ring-indigo-400"
								placeholder="Enter your username"
								bind:value={username}
								disabled={isLoading}
								autocomplete="username"
							/>
						</div>
						<div>
							<label
								for="password"
								class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>
								Password
							</label>
							<input
								id="password"
								name="password"
								type="password"
								required
								class="mt-2 block w-full rounded-lg border border-gray-200 bg-white px-4 py-3 text-base text-gray-900 placeholder-gray-500 transition-colors focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100 dark:placeholder-gray-500 dark:focus:border-indigo-400 dark:focus:ring-indigo-400"
								placeholder="Enter your password"
								bind:value={password}
								disabled={isLoading}
								autocomplete="current-password"
							/>
						</div>
					</div>

					<!-- Login button -->
					<div class="mt-8 flex justify-end">
						<button
							type="submit"
							disabled={isLoading}
							class="inline-flex items-center justify-center gap-2 rounded-lg bg-indigo-600 px-8 py-3 text-base font-medium text-white transition-all hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-gray-900 cursor-pointer"
						>
							{#if isLoading}
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
								Signing in...
							{:else}
								Sign in
							{/if}
						</button>
					</div>
				</form>
			</div>
		</div>
	</div>
</div>
