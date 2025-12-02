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

	import { updateCredentials, currentUser } from "$lib/stores/authStore";
	import { get } from "svelte/store";
	import { toastStore } from "$lib/stores/toastStore";

	let currentPassword = "";
	let newUsername = "";
	let newPassword = "";
	let isSubmitting = false;
	let error = "";
	let success = false;

	$: username = $currentUser;

	async function handleSubmit() {
		error = "";
		success = false;

		// Validation
		if (!currentPassword) {
			error = "Current password is required";
			return;
		}

		if (!newUsername && !newPassword) {
			error = "At least one of new username or new password must be provided";
			return;
		}

		isSubmitting = true;

		try {
			await updateCredentials(newUsername || null, newPassword || null, currentPassword);
			toastStore.success("Credentials updated successfully");
			success = true;
			currentPassword = "";
			newUsername = "";
			newPassword = "";
		} catch (err) {
			error = err instanceof Error ? err.message : "Failed to update credentials";
			toastStore.error(error);
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="space-y-6">
	<div>
		<h3 class="text-base font-semibold text-gray-900 dark:text-gray-100">Credentials</h3>
		<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
			Update your username and password. You must provide your current password to make changes.
		</p>
	</div>

	{#if error}
		<div
			class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400"
			role="alert"
		>
			{error}
		</div>
	{/if}

	{#if success}
		<div
			class="rounded-lg border border-green-200 bg-green-50 p-4 text-sm text-green-800 dark:border-green-800 dark:bg-green-900/20 dark:text-green-400"
			role="alert"
		>
			Credentials updated successfully.
		</div>
	{/if}

	<form on:submit|preventDefault={handleSubmit} class="space-y-4">
		<div>
			<label
				for="current-password"
				class="block text-sm font-medium text-gray-700 dark:text-gray-300"
			>
				Current Password
			</label>
			<input
				id="current-password"
				type="password"
				required
				bind:value={currentPassword}
				disabled={isSubmitting}
				class="mt-1 block w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 placeholder-gray-500 focus:border-blue-500 focus:outline-none focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400 dark:focus:border-blue-400 dark:focus:ring-blue-400 sm:text-sm"
				autocomplete="current-password"
			/>
		</div>

		<div>
			<label for="new-username" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
				New Username (optional)
			</label>
			<input
				id="new-username"
				type="text"
				bind:value={newUsername}
				disabled={isSubmitting}
				placeholder={username || "admin"}
				class="mt-1 block w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 placeholder-gray-500 focus:border-blue-500 focus:outline-none focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400 dark:focus:border-blue-400 dark:focus:ring-blue-400 sm:text-sm"
				autocomplete="username"
			/>
		</div>

		<div>
			<label for="new-password" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
				New Password (optional)
			</label>
			<input
				id="new-password"
				type="password"
				bind:value={newPassword}
				disabled={isSubmitting}
				class="mt-1 block w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 placeholder-gray-500 focus:border-blue-500 focus:outline-none focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400 dark:focus:border-blue-400 dark:focus:ring-blue-400 sm:text-sm"
				autocomplete="new-password"
			/>
		</div>

		<div class="flex justify-end">
			<button
				type="submit"
				disabled={isSubmitting}
				class="rounded-lg border border-transparent bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-500 px-4 py-2 text-sm font-medium text-white transition hover:scale-105 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-gray-900 cursor-pointer"
			>
				{isSubmitting ? "Updating..." : "Update Credentials"}
			</button>
		</div>
	</form>
</div>
