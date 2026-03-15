<script lang="ts">
    import { type IConfig, useConfig } from "$lib/contexts/ConfigProvider.svelte";

    const { toggleFullscreen, changeSidebarTheme, changeDirection, changeTheme, reset, changeFontFamily } = useConfig();

    const fontFamilies: { value: IConfig["fontFamily"]; label: string; className?: string }[] = [
        {
            value: "dm-sans",
            label: "DM Sans",
            className: "group-[[data-font-family=dm-sans]]/html:bg-base-200",
        },
        {
            value: "wix",
            label: "Wix",
            className: "group-[[data-font-family=wix]]/html:bg-base-200",
        },
        {
            value: "inclusive",
            label: "Inclusive",
            className:
                "group-[[data-font-family=inclusive]]/html:bg-base-200 group-[:not([data-font-family])]/html:bg-base-200",
        },
        {
            value: "ar-one",
            label: "AR One",
            className: "group-[[data-font-family=ar-one]]/html:bg-base-200",
        },
    ];
</script>

<div class="drawer drawer-end">
    <input id="layout-rightbar-drawer" type="checkbox" class="drawer-toggle" />
    <div class="drawer-side z-[50]">
        <label for="layout-rightbar-drawer" aria-label="close sidebar" class="drawer-overlay" aria-hidden="true"
        ></label>
        <div class="bg-base-100 text-base-content h-full w-72 sm:w-96">
            <div class="bg-base-200 flex justify-between px-5 py-4">
                <p class="text-lg font-medium">Setting</p>
                <div class="inline-flex gap-1">
                    <button class="btn-ghost btn btn-sm btn-circle relative" onclick={reset} aria-label="Reset">
                        <span class="iconify lucide--rotate-cw size-5"></span>
                        <span
                            class="bg-error absolute end-0.5 top-0.5 rounded-full p-0 opacity-0 transition-all group-data-[changed]/html:p-[2px] group-data-[changed]/html:opacity-100"
                        ></span>
                    </button>
                    <button class="btn btn-ghost btn-sm btn-circle" onclick={toggleFullscreen} aria-label="Full Screen">
                        <span class="iconify lucide--minimize hidden size-5 group-data-[fullscreen]/html:inline"></span>
                        <span class="iconify lucide--fullscreen inline size-5 group-data-[fullscreen]/html:hidden"
                        ></span>
                    </button>
                    <label
                        for="layout-rightbar-drawer"
                        aria-label="close sidebar"
                        aria-hidden="true"
                        class="btn btn-ghost btn-sm btn-circle">
                        <span class="iconify lucide--x size-5"></span>
                    </label>
                </div>
            </div>
            <div class="p-5">
                <p class="font-medium">Theme</p>
                <div class="mt-3 grid grid-cols-3 gap-3">
                    <div
                        data-theme="light"
                        class="rounded-box group relative cursor-pointer"
                        onclick={() => changeTheme("light")}>
                        <div class="bg-base-200 rounded-box pt-5 pb-3 text-center">
                            <div class="flex items-center justify-center gap-1">
                                <span class="rounded-box bg-primary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-secondary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-accent h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-success h-6 w-2 sm:w-3"></span>
                            </div>
                            <p class="mt-1.5 text-sm capitalize sm:text-base">Light</p>
                        </div>
                        <span
                            class="bg-primary text-primary-content absolute end-2 top-2 rounded-full p-0 opacity-0 transition-all group-data-[theme=light]/html:p-1 group-data-[theme=light]/html:opacity-100"
                        ></span>
                    </div>
                    <div
                        data-theme="contrast"
                        class="rounded-box group relative cursor-pointer"
                        onclick={() => changeTheme("contrast")}>
                        <div class="bg-base-200 rounded-box pt-5 pb-3 text-center">
                            <div class="flex items-center justify-center gap-1">
                                <span class="rounded-box bg-primary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-secondary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-accent h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-success h-6 w-2 sm:w-3"></span>
                            </div>
                            <p class="mt-1.5 text-sm capitalize sm:text-base">Contrast</p>
                        </div>
                        <span
                            class="bg-primary text-primary-content absolute end-2 top-2 rounded-full p-0 opacity-0 transition-all group-data-[theme=contrast]/html:p-1 group-data-[theme=contrast]/html:opacity-100"
                        ></span>
                    </div>
                    <div
                        data-theme="material"
                        class="rounded-box group relative cursor-pointer"
                        onclick={() => changeTheme("material")}>
                        <div class="bg-base-200 rounded-box pt-5 pb-3 text-center">
                            <div class="flex items-center justify-center gap-1">
                                <span class="rounded-box bg-primary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-secondary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-accent h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-success h-6 w-2 sm:w-3"></span>
                            </div>
                            <p class="mt-1.5 text-sm capitalize sm:text-base">Material</p>
                        </div>
                        <span
                            class="bg-primary text-primary-content absolute end-2 top-2 rounded-full p-0 opacity-0 transition-all group-data-[theme=material]/html:p-1 group-data-[theme=material]/html:opacity-100"
                        ></span>
                    </div>

                    <div
                        data-theme="dark"
                        class="rounded-box group relative cursor-pointer"
                        onclick={() => changeTheme("dark")}>
                        <div class="bg-base-200 rounded-box pt-5 pb-3 text-center">
                            <div class="flex items-center justify-center gap-1">
                                <span class="rounded-box bg-primary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-secondary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-accent h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-success h-6 w-2 sm:w-3"></span>
                            </div>
                            <p class="mt-1.5 text-sm capitalize sm:text-base">Dark</p>
                        </div>
                        <span
                            class="bg-primary text-primary-content absolute end-2 top-2 rounded-full p-0 opacity-0 transition-all group-data-[theme=dark]/html:p-1 group-data-[theme=dark]/html:opacity-100"
                        ></span>
                    </div>

                    <div
                        data-theme="dim"
                        class="rounded-box group relative cursor-pointer"
                        onclick={() => changeTheme("dim")}>
                        <div class="bg-base-200 rounded-box pt-5 pb-3 text-center">
                            <div class="flex items-center justify-center gap-1">
                                <span class="rounded-box bg-primary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-secondary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-accent h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-success h-6 w-2 sm:w-3"></span>
                            </div>
                            <p class="mt-1.5 text-sm capitalize sm:text-base">Dim</p>
                        </div>
                        <span
                            class="bg-primary text-primary-content absolute end-2 top-2 rounded-full p-0 opacity-0 transition-all group-data-[theme=dim]/html:p-1 group-data-[theme=dim]/html:opacity-100"
                        ></span>
                    </div>

                    <div
                        data-theme="material-dark"
                        class="rounded-box group relative cursor-pointer"
                        onclick={() => changeTheme("material-dark")}>
                        <div class="bg-base-200 rounded-box pt-5 pb-3 text-center">
                            <div class="flex items-center justify-center gap-1">
                                <span class="rounded-box bg-primary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-secondary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-accent h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-success h-6 w-2 sm:w-3"></span>
                            </div>
                            <p class="mt-1.5 text-xs capitalize sm:text-sm">Material Dark</p>
                        </div>
                        <span
                            class="bg-primary text-primary-content absolute end-2 top-2 rounded-full p-0 opacity-0 transition-all group-data-[theme=material-dark]/html:p-1 group-data-[theme=material-dark]/html:opacity-100"
                        ></span>
                    </div>

                    <div class="rounded-box group relative cursor-pointer" onclick={() => changeTheme("system")}>
                        <div class="bg-base-200 rounded-box pt-5 pb-3 text-center">
                            <div class="flex items-center justify-center gap-1">
                                <span class="rounded-box bg-primary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-secondary h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-accent h-6 w-2 sm:w-3"></span>
                                <span class="rounded-box bg-success h-6 w-2 sm:w-3"></span>
                            </div>
                            <p class="mt-1.5 text-sm capitalize sm:text-base">System</p>
                        </div>
                        <span
                            class="bg-primary text-primary-content absolute end-2 top-2 rounded-full p-0 opacity-0 transition-all group-[:not([data-theme])]/html:p-1 group-[:not([data-theme])]/html:opacity-100"
                        ></span>
                    </div>
                </div>
                <div
                    class="pointer-events-none opacity-50 group-data-[theme=contrast]/html:pointer-events-auto group-data-[theme=contrast]/html:opacity-100 group-data-[theme=light]/html:pointer-events-auto group-data-[theme=light]/html:opacity-100">
                    <p class="mt-6 font-medium">
                        Sidebar
                        <span
                            class="ms-1 inline text-xs group-data-[theme=contrast]/html:hidden group-data-[theme=light]/html:hidden md:text-sm">
                            (*Only available in light, contrast themes)
                        </span>
                    </p>
                    <div class="mt-3 grid grid-cols-2 gap-3">
                        <div
                            class="border-base-300 hover:bg-base-200 rounded-box group-data-[sidebar-theme=light]/html:bg-base-200 inline-flex cursor-pointer items-center justify-center gap-2 border p-2"
                            onclick={() => changeSidebarTheme("light")}>
                            <span class="iconify lucide--sun size-4.5"></span>
                            Light
                        </div>
                        <div
                            class="border-base-300 hover:bg-base-200 rounded-box group-data-[sidebar-theme=dark]/html:bg-base-200 inline-flex cursor-pointer items-center justify-center gap-2 border p-2"
                            onclick={() => changeSidebarTheme("dark")}>
                            <span class="iconify lucide--moon size-4.5"></span>
                            Dark
                        </div>
                    </div>
                </div>
                <p class="mt-6 font-medium">Font Family</p>
                <div class="mt-3 grid grid-cols-2 gap-3">
                    {#each fontFamilies as item, index (index)}
                        <div
                            class="border-base-300 hover:bg-base-200 rounded-box inline-flex cursor-pointer items-center justify-center gap-2 border p-2 {item.className}"
                            onclick={() => changeFontFamily(item.value)}>
                            <p data-font-family={item.value} class="font-sans">{item.label}</p>
                        </div>
                    {/each}
                </div>
                <p class="mt-6 font-medium">Direction</p>
                <div class="mt-3 grid grid-cols-2 gap-3">
                    <div
                        class="border-base-300 hover:bg-base-200 rounded-box group-[[dir=ltr]]/html:bg-base-200 group-[:not([dir])]/html:bg-base-200 inline-flex cursor-pointer items-center justify-center gap-2 border p-2"
                        onclick={() => changeDirection("ltr")}>
                        <span class="iconify lucide--pilcrow-left size-4.5"></span>
                        <span class="hidden sm:inline">Left to Right</span>
                        <span class="inline sm:hidden">LTR</span>
                    </div>
                    <div
                        class="border-base-300 hover:bg-base-200 rounded-box group-[[dir=rtl]]/html:bg-base-200 inline-flex cursor-pointer items-center justify-center gap-2 border p-2"
                        onclick={() => changeDirection("rtl")}>
                        <span class="iconify lucide--pilcrow-right size-4.5"></span>
                        <span class="hidden sm:inline">Right to Right</span>
                        <span class="inline sm:hidden">RTL</span>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>
