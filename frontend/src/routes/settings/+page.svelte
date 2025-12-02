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

	import { onMount, onDestroy, getContext } from "svelte";
	import type { PageData } from "./$types";
	import SettingsView from "$lib/components/settings/SettingsView.svelte";
	import RulesSection from "$lib/components/settings/RulesSection.svelte";
	import NotificationSettingsSection from "$lib/components/settings/NotificationSettingsSection.svelte";
	import ThemeSettingsSection from "$lib/components/settings/ThemeSettingsSection.svelte";
	import SyncOlderNotificationsSection from "$lib/components/settings/SyncOlderNotificationsSection.svelte";
	import { registerListShortcuts } from "$lib/keyboard/listShortcuts";
	import { registerCommand } from "$lib/keyboard/commandRegistry";

	export let data: PageData;

	// Get layout context to access command palette
	const layoutContext = getContext("layoutFunctions") as any;

	// Command palette handlers for shortcuts
	function openCommandPaletteForView(): boolean {
		const commandPalette = layoutContext?.getCommandPalette();
		if (!commandPalette) {
			return false;
		}
		commandPalette.openWithCommand("view");
		return true;
	}

	function openCommandPaletteEmpty(): boolean {
		const commandPalette = layoutContext?.getCommandPalette();
		if (!commandPalette) {
			return false;
		}
		commandPalette.focusInput();
		return true;
	}

	function toggleShortcutsModal(): boolean {
		return layoutContext?.toggleShortcutsModal() ?? false;
	}

	function isAnyDialogOpen(): boolean {
		return layoutContext?.isAnyDialogOpen() ?? false;
	}

	let unregisterShortcuts: (() => void) | null = null;

	onMount(() => {
		// Register commands for shortcuts
		registerCommand("openPaletteView", () => openCommandPaletteForView());
		registerCommand("openPaletteEmpty", () => openCommandPaletteEmpty());
		registerCommand("toggleShortcutsModal", () => toggleShortcutsModal());

		// Register shortcuts - only Cmd+K and Cmd+Shift+K are needed for settings
		unregisterShortcuts = registerListShortcuts({
			getDetailOpen: () => false,
			isAnyDialogOpen,
			isFilterDropdownOpen: () => false,
		});

		return () => {
			if (unregisterShortcuts) {
				unregisterShortcuts();
			}
		};
	});

	onDestroy(() => {
		if (unregisterShortcuts) {
			unregisterShortcuts();
		}
	});
</script>

<SettingsView title="" description="">
	<div class="space-y-8">
		<ThemeSettingsSection />
		<div class="border-t border-gray-200 dark:border-gray-800 pt-8">
			<NotificationSettingsSection />
		</div>
		<div class="border-t border-gray-200 dark:border-gray-800 pt-8">
			<RulesSection rules={data.rules} tags={data.tags} />
		</div>
		<div class="border-t border-gray-200 dark:border-gray-800 pt-8">
			<SyncOlderNotificationsSection />
		</div>
	</div>
</SettingsView>
