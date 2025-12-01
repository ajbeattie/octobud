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

	export let title: string;
	export let description: string;
</script>

<div class="flex-1 flex flex-col overflow-y-auto bg-white dark:bg-gray-950">
	<!-- Content -->
	<div class="flex-1">
		<div class="max-w-5xl mx-auto px-6 py-6">
			{#if (title && title.trim()) || (description && description.trim())}
				<!-- Header (only shown if title or description provided) -->
				<div
					class="border-b border-gray-200 dark:border-gray-800 bg-white/50 dark:bg-gray-950/50 mb-6 -mx-6 px-6 py-6"
				>
					{#if title && title.trim()}
						<h1 class="text-xl font-semibold text-gray-900 dark:text-gray-100">{title}</h1>
					{/if}
					{#if description && description.trim()}
						<p class="text-sm text-gray-600 dark:text-gray-400 mt-1">{description}</p>
					{/if}
				</div>
			{/if}
			<slot />
		</div>
	</div>
</div>
