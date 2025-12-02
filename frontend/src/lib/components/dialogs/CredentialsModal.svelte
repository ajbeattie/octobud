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

	import Modal from "$lib/components/shared/Modal.svelte";
	import { updateCredentials, currentUser } from "$lib/stores/authStore";
	import { get } from "svelte/store";
	import { toastStore } from "$lib/stores/toastStore";
	import { validatePasswordStrength } from "$lib/utils/password";
	import { validateUsername } from "$lib/utils/username";

	export let open = false;
	export let onClose: () => void = () => {};

	let currentPassword = "";
	let newUsername = "";
	let newPassword = "";
	let isSubmitting = false;
	let error = "";
	let success = false;

	$: username = $currentUser;

	function handleClose() {
		// Reset form state when closing
		currentPassword = "";
		newUsername = "";
		newPassword = "";
		error = "";
		success = false;
		onClose();
	}

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

		// Validate username format if new username is provided
		if (newUsername) {
			const usernameValidation = validateUsername(newUsername);
			if (!usernameValidation.valid) {
				error = usernameValidation.error || "Invalid username";
				return;
			}
		}

		// Validate password strength if new password is provided
		if (newPassword) {
			const passwordValidation = validatePasswordStrength(newPassword);
			if (!passwordValidation.valid) {
				error = passwordValidation.error || "Invalid password";
				return;
			}
		}

		isSubmitting = true;

		try {
			await updateCredentials(newUsername || null, newPassword || null, currentPassword);
			toastStore.success("Credentials updated successfully");
			success = true;
			// Close modal after a brief delay to show success message
			setTimeout(() => {
				handleClose();
			}, 1000);
		} catch (err) {
			error = err instanceof Error ? err.message : "Failed to update credentials";
			toastStore.error(error);
		} finally {
			isSubmitting = false;
		}
	}
</script>

<Modal {open} title="Update Credentials" size="md" {onClose}>
	<div class="space-y-6">
		<p class="text-sm text-gray-600 dark:text-gray-400">
			Update your username and password. You must provide your current password to make changes.
		</p>

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
				<label
					for="new-username"
					class="block text-sm font-medium text-gray-700 dark:text-gray-300"
				>
					New Username (optional)
				</label>
				<input
					id="new-username"
					type="text"
					bind:value={newUsername}
					disabled={isSubmitting}
					placeholder={username || "octobud"}
					class="mt-1 block w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 placeholder-gray-500 focus:border-blue-500 focus:outline-none focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400 dark:focus:border-blue-400 dark:focus:ring-blue-400 sm:text-sm"
					autocomplete="username"
					minlength="3"
					maxlength="64"
				/>
				{#if newUsername}
					{@const validation = validateUsername(newUsername)}
					{#if !validation.valid}
						<p class="mt-1 text-sm text-red-600 dark:text-red-400">
							{validation.error}
						</p>
					{/if}
				{/if}
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
					Username must be 3-64 characters. Allowed: letters, numbers, underscores, dots, hyphens,
					and @.
				</p>
			</div>

			<div>
				<label
					for="new-password"
					class="block text-sm font-medium text-gray-700 dark:text-gray-300"
				>
					New Password (optional)
				</label>
				<input
					id="new-password"
					type="password"
					bind:value={newPassword}
					disabled={isSubmitting}
					class="mt-1 block w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 placeholder-gray-500 focus:border-blue-500 focus:outline-none focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400 dark:focus:border-blue-400 dark:focus:ring-blue-400 sm:text-sm"
					autocomplete="new-password"
					minlength="8"
					maxlength="128"
				/>
				{#if newPassword}
					{@const validation = validatePasswordStrength(newPassword)}
					{#if !validation.valid}
						<p class="mt-1 text-sm text-red-600 dark:text-red-400">
							{validation.error}
						</p>
					{/if}
				{/if}
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
					Password must be between 8 and 128 characters long.
				</p>
			</div>

			<div class="flex items-center justify-end gap-3 pt-2">
				<button
					type="button"
					on:click={handleClose}
					disabled={isSubmitting}
					class="rounded-full border border-transparent px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 transition hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
				>
					Cancel
				</button>
				<button
					type="submit"
					disabled={isSubmitting}
					class="rounded-full bg-indigo-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-indigo-700 disabled:opacity-50 disabled:hover:bg-indigo-600 cursor-pointer"
				>
					{isSubmitting ? "Updating..." : "Update Credentials"}
				</button>
			</div>
		</form>
	</div>
</Modal>
