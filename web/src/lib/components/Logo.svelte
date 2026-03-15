<script lang="ts">
    type ILogo = {
        className?: string;
    };

    let { className }: ILogo = $props();
</script>

<div class={`flex flex-col items-center ${className ?? ""}`}>
    <span class="text-lg font-bold text-primary tracking-widest leading-tight">HIGHWATER</span>
    <span class="text-lg font-bold text-primary tracking-widest leading-tight">HUB</span>
    <span class="text-[0.6rem] opacity-40 tracking-[0.2em]">FLOOD FAMILY HQ</span>
</div>
