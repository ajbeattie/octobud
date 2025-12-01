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

	import NotificationListSection from "./NotificationListSection.svelte";
	import InlineDetailView from "$lib/components/detail/InlineDetailView.svelte";
	import type { Notification, NotificationDetail } from "$lib/api/types";

	import type { TimelineController } from "$lib/state/timelineController";

	// List view props
	export let hasActiveFilters: boolean;
	export let totalCount: number;
	export let pageRangeStart: number;
	export let pageRangeEnd: number;
	export let items: Notification[];
	export let isLoading: boolean;
	export let selectionEnabled: boolean;
	export let individualSelectionDisabled: boolean;
	export let detailNotificationId: string | null;
	export let selectionMap: Map<string, boolean>;

	// Detail view props
	export let detailOpen: boolean;
	export let currentDetailNotification: Notification | null;
	export let currentDetail: NotificationDetail | null;
	export let detailLoading: boolean;
	export let detailShowingStaleData: boolean;
	export let detailIsRefreshing: boolean;
	export let timelineController: TimelineController | undefined;

	// Scroll restoration
	export let initialScrollPosition: number = 0;

	// Export list section ref for parent access
	let listSectionComponent: any = null;
	export { listSectionComponent };
</script>

{#if detailOpen && currentDetailNotification}
	<!-- Gmail-style inline detail view -->
	<InlineDetailView
		detail={currentDetail}
		loading={detailLoading}
		showingStaleData={detailShowingStaleData}
		isRefreshing={detailIsRefreshing}
		isSplitView={false}
		{timelineController}
		markingRead={false}
		archiving={false}
	/>
{:else}
	<!-- List view -->
	<NotificationListSection
		bind:this={listSectionComponent}
		{hasActiveFilters}
		{totalCount}
		{pageRangeStart}
		{pageRangeEnd}
		{items}
		{isLoading}
		{selectionEnabled}
		{individualSelectionDisabled}
		{detailNotificationId}
		isSplitView={false}
		{selectionMap}
		{initialScrollPosition}
	/>
{/if}
