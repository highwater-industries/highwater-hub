<script lang="ts">
    // @ts-ignore
    import { afterNavigate } from "$app/navigation";
    // @ts-ignore
    import { page } from "$app/state";
    import Logo from "$lib/components/Logo.svelte";
    import { useConfig } from "$lib/contexts/ConfigProvider.svelte";
    import SimpleBar from "simplebar";
    import "simplebar/dist/simplebar.min.css";
    import SidebarMenuItem, { type ISidebarMenuItem } from "./SidebarMenuItem.svelte";
    import { getActivatedItemParentKeys } from "./helpers";

    let { menuItems }: { menuItems: ISidebarMenuItem[] } = $props();

    const { config } = useConfig();

    let activatedParents = $state(new Set(getActivatedItemParentKeys(menuItems, page.url.pathname)));
    let scrollRef: HTMLDivElement;
    let simplebar: SimpleBar | undefined;

    afterNavigate(() => {
        activatedParents = new Set(getActivatedItemParentKeys(menuItems, page.url.pathname));
        setTimeout(() => {
            const contentElement = simplebar?.getContentElement();
            const scrollElement = simplebar?.getScrollElement();
            if (contentElement) {
                const activatedItem = contentElement.querySelector<HTMLElement>(".active");
                const top = activatedItem?.getBoundingClientRect().top;
                if (activatedItem && scrollElement && top && top !== 0) {
                    scrollElement.scrollTo({ top: scrollElement.scrollTop + top - 300, behavior: "smooth" });
                }
            }
        }, 100);

        if (window.innerWidth <= 64 * 16) {
            const sidebarTrigger = document.querySelector<HTMLInputElement>("#layout-sidebar-toggle-trigger");
            if (sidebarTrigger) {
                sidebarTrigger.checked = false;
            }
        }
    });

    $effect(() => {
        simplebar = new SimpleBar(scrollRef);
    });
</script>

<input class="hidden" id="layout-sidebar-toggle-trigger" type="checkbox" aria-label="Toggle layout sidebar" />
<input type="checkbox" id="layout-sidebar-hover-trigger" class="hidden" aria-label="Dense layout sidebar" />
<div id="layout-sidebar-hover" class="bg-base-300 h-screen w-1"></div>
<div
    id="layout-sidebar"
    class="sidebar-menu flex flex-col"
    data-theme={$config.sidebarTheme === "dark" && ["light", "contrast"].includes($config.theme) ? "dark" : undefined}>
    <div class="flex h-16 min-h-16 items-center justify-between gap-3 ps-5 pe-4">
        <a href="/dashboards/ecommerce">
            <Logo />
        </a>
        <label
            for="layout-sidebar-hover-trigger"
            title="Toggle sidebar hover"
            class="btn btn-circle btn-ghost btn-sm text-base-content/50 relative max-lg:hidden">
            <span
                class="iconify lucide--panel-left-close absolute size-4.5 opacity-100 transition-all duration-300 group-has-[[id=layout-sidebar-hover-trigger]:checked]/html:opacity-0" />
            <span
                class="iconify lucide--panel-left-dashed absolute size-4.5 opacity-0 transition-all duration-300 group-has-[[id=layout-sidebar-hover-trigger]:checked]/html:opacity-100" />
        </label>
    </div>

    <div class="relative min-h-0 grow">
        <div bind:this={scrollRef} class="size-full">
            <div class="mb-3 space-y-0.5 px-2.5">
                {#each menuItems as item, index (index)}
                    <SidebarMenuItem {...item} activated={activatedParents} />
                {/each}
            </div>
        </div>
        <div
            class="from-base-100/60 absolute start-0 end-0 bottom-0 h-7 bg-linear-to-t to-transparent pointer-events-none">
        </div>
    </div>

    <div class="mb-2">
        <a target="_blank" class="group rounded-box relative mx-2.5 block gap-3" href="/components">
            <div
                class="rounded-box absolute inset-0 bg-gradient-to-r from-transparent to-transparent transition-opacity duration-300 group-hover:opacity-0">
            </div>
            <div
                class="from-primary to-secondary rounded-box absolute inset-0 bg-gradient-to-r opacity-0 transition-opacity duration-300 group-hover:opacity-100">
            </div>
            <div class="relative flex h-10 items-center gap-3 px-3">
                <i
                    class="iconify lucide--shapes text-primary size-4.5 transition-all duration-300 group-hover:text-white"
                ></i>
                <p
                    class="from-primary to-secondary bg-gradient-to-r bg-clip-text font-medium text-transparent transition-all duration-300 group-hover:text-white">
                    Components
                </p>
                <i
                    class="iconify lucide--chevron-right text-secondary ms-auto size-4.5 transition-all duration-300 group-hover:text-white"
                ></i>
            </div>
        </a>
        <hr class="border-base-300 my-2 border-dashed" />
        <div class="dropdown dropdown-top dropdown-end w-full">
            <div
                tabindex="0"
                role="button"
                class="bg-base-200 hover:bg-base-300 rounded-box mx-2 mt-0 flex cursor-pointer items-center gap-2.5 px-3 py-2 transition-all">
                <div class="avatar">
                    <div class="bg-base-200 mask mask-squircle w-8">
                        <img src="/images/avatars/1.png" alt="Avatar" />
                    </div>
                </div>
                <div class="grow -space-y-0.5">
                    <p class="text-sm font-medium">Denish N</p>
                    <p class="text-base-content/60 text-xs">@withden</p>
                </div>
                <span class="iconify lucide--chevrons-up-down text-base-content/60 size-4"></span>
            </div>
            <ul
                role="menu"
                tabindex="0"
                class="dropdown-content menu bg-base-100 rounded-box shadow-base-content/4 mb-1 w-48 p-1 shadow-[0px_-10px_40px_0px]">
                <li>
                    <a href="/pages/settings">
                        <span class="iconify lucide--user size-4"></span><span>My Profile</span>
                    </a>
                </li>
                <li>
                    <a href="/pages/settings">
                        <span class="iconify lucide--settings size-4"></span><span>Settings</span>
                    </a>
                </li>
                <li>
                    <a href="/pages/get-help">
                        <span class="iconify lucide--help-circle size-4"></span><span>Help</span>
                    </a>
                </li>
                <li>
                    <div><span class="iconify lucide--bell size-4"></span><span>Notification</span></div>
                </li>
                <li>
                    <div>
                        <span class="iconify lucide--arrow-left-right size-4"></span><span>Switch Account</span>
                    </div>
                </li>
            </ul>
        </div>
    </div>
</div>

<label for="layout-sidebar-toggle-trigger" id="layout-sidebar-backdrop"></label>
