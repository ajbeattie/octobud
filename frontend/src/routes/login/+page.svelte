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
	import { login, isAuthenticated } from "$lib/stores/authStore";
	import { get } from "svelte/store";

	let username = "";
	let password = "";
	let error = "";
	let isLoading = false;

	onMount(async () => {
		// Redirect if already authenticated
		if (get(isAuthenticated)) {
			await goto(resolve("/views/inbox"));
		}
	});

	async function handleSubmit() {
		error = "";
		isLoading = true;

		try {
			await login(username, password);
			await goto(resolve("/views/inbox"));
		} catch (err) {
			error = err instanceof Error ? err.message : "Login failed";
		} finally {
			isLoading = false;
		}
	}
</script>

<div
	class="flex min-h-screen items-center justify-center bg-gray-50 px-4 py-12 dark:bg-gray-900 sm:px-6 lg:px-8"
>
	<div class="w-full max-w-md space-y-8">
		<div class="text-center">
			<div
				class="mx-auto flex h-16 w-16 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-500 text-indigo-50 shadow-soft"
			>
				<img src="/baby_octo.svg" alt="Octobud icon" class="h-10 w-10" />
			</div>
			<h2 class="mt-6 text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
				Sign in to Octobud
			</h2>
			<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
				Default credentials: <span class="font-mono">octobud</span> /
				<span class="font-mono">octobud</span>
			</p>
		</div>

		<form class="mt-8 space-y-6" on:submit|preventDefault={handleSubmit}>
			{#if error}
				<div
					class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400"
					role="alert"
				>
					{error}
				</div>
			{/if}

			<div class="space-y-4 rounded-md shadow-sm">
				<div>
					<label for="username" class="sr-only">Username</label>
					<input
						id="username"
						name="username"
						type="text"
						required
						class="relative block w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 placeholder-gray-500 focus:z-10 focus:border-blue-500 focus:outline-none focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400 dark:focus:border-blue-400 dark:focus:ring-blue-400 sm:text-sm"
						placeholder="Username"
						bind:value={username}
						disabled={isLoading}
						autocomplete="username"
					/>
				</div>
				<div>
					<label for="password" class="sr-only">Password</label>
					<input
						id="password"
						name="password"
						type="password"
						required
						class="relative block w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-gray-900 placeholder-gray-500 focus:z-10 focus:border-blue-500 focus:outline-none focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400 dark:focus:border-blue-400 dark:focus:ring-blue-400 sm:text-sm"
						placeholder="Password"
						bind:value={password}
						disabled={isLoading}
						autocomplete="current-password"
					/>
				</div>
			</div>

			<div>
				<button
					type="submit"
					disabled={isLoading}
					class="group relative flex w-full justify-center rounded-lg border-transparent bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-500 px-4 py-2 text-sm font-medium text-white transition hover:scale-105 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-gray-900"
				>
					{isLoading ? "Signing in..." : "Sign in"}
				</button>
			</div>
		</form>
	</div>
</div>
