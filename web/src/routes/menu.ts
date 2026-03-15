import type { ISidebarMenuItem } from "$lib/components/admin-layout/SidebarMenuItem.svelte";

export const appMenuItems: ISidebarMenuItem[] = [
    {
        id: "overview-label",
        isTitle: true,
        label: "Overview",
    },
    {
        id: "dashboard",
        icon: "lucide--layout-dashboard",
        label: "Dashboard",
        url: "/",
    },
    {
        id: "fitness-label",
        isTitle: true,
        label: "Fitness",
    },
    {
        id: "fitness",
        icon: "lucide--dumbbell",
        label: "Workouts",
        url: "/fitness",
    },
    {
        id: "fitness-progress",
        icon: "lucide--trending-up",
        label: "Progress",
        url: "/fitness/progress",
    },
    {
        id: "sports-label",
        isTitle: true,
        label: "NFL",
    },
    {
        id: "players",
        icon: "lucide--users",
        label: "Players",
        url: "/players",
    },
    {
        id: "stats",
        icon: "lucide--bar-chart-3",
        label: "Stats",
        url: "/stats",
    },
    {
        id: "games",
        icon: "lucide--calendar",
        label: "Games",
        url: "/games",
    },
    {
        id: "rankings",
        icon: "lucide--trophy",
        label: "Rankings",
        url: "/rankings",
    },
    {
        id: "system-label",
        isTitle: true,
        label: "System",
    },
    {
        id: "data",
        icon: "lucide--database",
        label: "Data Management",
        url: "/data",
    },
    {
        id: "media",
        icon: "lucide--play-circle",
        label: "Media",
        url: "/media",
    },
];
