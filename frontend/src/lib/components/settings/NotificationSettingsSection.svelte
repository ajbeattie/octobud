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
	import { getNotificationSettingsStore } from "$lib/stores/notificationSettings";
	import {
		requestNotificationPermission,
		getNotificationPermission,
		sendNotificationSettingToSW,
		sendTestNotificationToSW,
	} from "$lib/utils/serviceWorkerRegistration";
	import { toastStore } from "$lib/stores/toastStore";
	import { browser } from "$app/environment";

	const settingsStore = getNotificationSettingsStore();
	const { notificationsEnabled } = settingsStore;

	let isSupported = false;
	let permission: NotificationPermission = "default";
	let isToggling = false;
	let isTestingNotification = false;

	onMount(() => {
		if (browser) {
			isSupported = "Notification" in window && "serviceWorker" in navigator;
			permission = getNotificationPermission();
		}
	});

	async function handleToggle(enabled: boolean) {
		if (isToggling) return;

		isToggling = true;

		try {
			if (enabled) {
				// Turning notifications ON - request permission if needed
				const currentPermission = getNotificationPermission();

				if (currentPermission === "denied") {
					toastStore.error(
						"Notification permission is denied. Please enable it in your browser settings."
					);
					// Don't enable the toggle if permission is denied
					settingsStore.setEnabled(false);
					return;
				}

				if (currentPermission === "default") {
					// Request permission
					const newPermission = await requestNotificationPermission();
					permission = newPermission; // Update local state

					if (newPermission !== "granted") {
						toastStore.error(
							"Notification permission is required to enable desktop notifications."
						);
						settingsStore.setEnabled(false);
						return;
					}

					toastStore.success("Desktop notifications enabled");
				} else {
					// Permission already granted
					permission = currentPermission; // Update local state
					toastStore.success("Desktop notifications enabled");
				}

				// Set enabled and notify service worker
				settingsStore.setEnabled(true);
				await sendNotificationSettingToSW(true);
			} else {
				// Turning notifications OFF
				settingsStore.setEnabled(false);
				await sendNotificationSettingToSW(false);
				toastStore.success("Desktop notifications disabled");
			}
		} catch (error) {
			console.error("Failed to toggle notifications:", error);
			toastStore.error("Failed to update notification settings");
		} finally {
			isToggling = false;
		}
	}

	async function handleTestNotification() {
		if (isTestingNotification) return;

		isTestingNotification = true;

		try {
			// Check if notifications are enabled
			if (!$notificationsEnabled) {
				toastStore.error("Please enable desktop notifications first");
				return;
			}

			// Check permission
			const currentPermission = getNotificationPermission();
			if (currentPermission !== "granted") {
				toastStore.error("Notification permission is not granted");
				return;
			}

			// Check if service worker is available
			if (!browser || !("serviceWorker" in navigator) || !navigator.serviceWorker.controller) {
				toastStore.error("Service worker is not available");
				return;
			}

			// Send test notification request to service worker
			await sendTestNotificationToSW();
			toastStore.success("Test notification sent");
		} catch (error) {
			console.error("Failed to send test notification:", error);
			toastStore.error("Failed to send test notification");
		} finally {
			isTestingNotification = false;
		}
	}

	$: canEnable = permission !== "denied";
	$: statusText =
		permission === "granted"
			? "Enabled"
			: permission === "denied"
				? "Permission denied"
				: "Permission not granted";
	$: canTestNotification =
		isSupported && permission === "granted" && $notificationsEnabled && !isTestingNotification;
</script>

<div class="space-y-4">
	<!-- Notification Toggle -->
	<div class="flex items-center justify-between py-3">
		<div class="flex-1">
			<div class="flex items-center gap-2">
				<span class="text-md font-medium text-gray-900 dark:text-gray-100">
					Enable Desktop Notifications
				</span>
				{#if !isSupported}
					<span class="text-xs text-gray-600 dark:text-gray-500"
						>(Not supported in this browser)</span
					>
				{/if}
			</div>
			<p class="text-xs text-gray-600 dark:text-gray-400 mt-1">
				Receive desktop notifications when new notifications arrive, even when the app is in the
				background.
				{#if permission === "denied"}
					<br />
					<span class="text-amber-600 dark:text-amber-500">
						Permission is denied. Enable notifications in your browser settings.
					</span>
				{/if}
			</p>
		</div>
		<label
			class="relative flex items-center cursor-pointer {!isSupported || isToggling
				? 'opacity-50 cursor-not-allowed'
				: ''}"
		>
			<input
				type="checkbox"
				checked={$notificationsEnabled}
				disabled={!isSupported || isToggling || !canEnable}
				on:change={(e) => handleToggle(e.currentTarget.checked)}
				class="sr-only peer"
			/>
			<div
				class="relative w-9 h-5 bg-gray-300 dark:bg-gray-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-600 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600 peer-disabled:opacity-50"
			></div>
		</label>
	</div>

	<!-- Status Info -->
	{#if isSupported}
		<div
			class="text-xs text-gray-600 dark:text-gray-500 bg-gray-50 dark:bg-gray-900/50 rounded-lg p-3 border border-gray-200 dark:border-gray-800"
		>
			<div
				class="flex items-center justify-between font-medium text-gray-700 dark:text-gray-400 mb-1"
			>
				<span>Status: {statusText}</span>
				{#if permission === "granted"}
					<button
						type="button"
						disabled={!canTestNotification}
						on:click={handleTestNotification}
						class="text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 hover:underline disabled:text-gray-400 dark:disabled:text-gray-600 disabled:cursor-not-allowed disabled:no-underline transition-colors cursor-pointer"
					>
						{isTestingNotification ? "Sending..." : "Send test"}
					</button>
				{/if}
			</div>
			<div class="space-y-1">
				<p>• Notifications work even when the browser tab is suspended</p>
				<p>• Only shows notifications for items that aren't archived or muted</p>
				<p>• Clicking a notification opens the app</p>
			</div>
		</div>
	{/if}
</div>
