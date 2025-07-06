<script lang="ts">
  import { EventsOn } from "@wails-runtime/runtime";
  import { onDestroy } from "svelte";
  import { getErrorHandler } from "./toast/toast";

  const onError = getErrorHandler();
  let isMaintaining = $state(false);

  const unregister = [
    EventsOn("maintenance-start", () => {
      isMaintaining = true;
    }),
    EventsOn("maintenance-end", (err) => {
      isMaintaining = false;
      if (err) {
        onError(err);
      }
    }),
  ];

  onDestroy(() => {
    unregister.forEach((rm) => rm());
  });
</script>

{#if isMaintaining}
  <div>Maintenance is running...</div>
{/if}

<style>
  div {
    position: fixed;
    background: var(--color-bg-elevated);
    color: var(--color-fg-elevated);
    z-index: var(--z-banner);

    bottom: 0;
    left: 0;
    right: 0;

    text-align: right;
    font-size: var(--text-sm);

    padding-inline: calc(var(--spacing) * 2);
    padding-block: calc(var(--spacing) * 1);
  }
</style>
