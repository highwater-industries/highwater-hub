<script lang="ts" module>
    import type { HTMLAnchorAttributes } from "svelte/elements";
    import SidebarMenuItem from "./SidebarMenuItem.svelte";
    import SidebarMenuItemBadges, { type ISidebarMenuItemBadges } from "./SidebarMenuItemBadges.svelte";

    export type ISidebarMenuItem = {
        id: string;
        icon?: string;
        label: string;
        isTitle?: boolean;
        url?: string;
        linkProp?: HTMLAnchorAttributes;
        children?: ISidebarMenuItem[];
    } & ISidebarMenuItemBadges;
</script>

<script lang="ts">
    let {
        id,
        url,
        children,
        icon,
        isTitle,
        badges,
        linkProp,
        label,
        activated,
    }: ISidebarMenuItem & { activated: Set<string> } = $props();
    let selected = $derived(activated.has(id));
</script>

{#if isTitle}
    <p class="menu-label px-2.5 pt-3 pb-1.5 first:pt-0">{label}</p>
{:else if !children}
    <a href={url ?? ""} class={`menu-item  ${selected && "active"}`} {...linkProp}>
        {#if icon}
            <span class={`iconify ${icon} size-4`}></span>
        {/if}
        <span class="grow">{label}</span>
        <SidebarMenuItemBadges {badges} />
    </a>
{:else}
    <div class="collapse group">
        <input
            aria-label="Sidemenu item trigger"
            type="checkbox"
            name="sidebar-menu-parent-item"
            defaultChecked={selected}
            class="peer" />
        <div class="collapse-title px-2.5 py-1.5">
            {#if icon}
                <span class={`iconify ${icon} size-4`} />
            {/if}
            <span class="grow">{label}</span>
            <SidebarMenuItemBadges {badges} />
            <span class="iconify lucide--chevron-right arrow-icon size-3.5" />
        </div>
        <div class="collapse-content ms-6.5 !p-0">
            <div class="mt-0.5 space-y-0.5">
                {#each children as item, index (index)}
                    <SidebarMenuItem {...item} {activated} />
                {/each}
            </div>
        </div>
    </div>
{/if}
