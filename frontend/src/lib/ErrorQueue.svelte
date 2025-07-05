<script lang="ts">
  import Button from "./components/Button.svelte";
  import { getErrorQueue } from "./error.svelte";

  const queue = getErrorQueue();

  const errorToShow = $derived(queue.errors[0]);
  const hasMultiple = $derived(queue.errors.length > 1);
</script>

{#if errorToShow}
  <dialog open>
    <div>
      {errorToShow}
    </div>

    <Button
      variant="secondary"
      onclick={() => {
        queue.errors.shift();
      }}
    >
      {#if hasMultiple}
        Next error
      {:else}
        Dismiss
      {/if}
    </Button>
  </dialog>
{/if}

<style>
  dialog {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    width: 100vw;
    padding: calc(var(--spacing) * 2);

    display: flex;
    justify-content: space-between;
    align-items: center;

    background-color: var(--color-bg-error);
    color: var(--color-fg-error);

    font-size: var(--text-sm);
    font-weight: var(--font-semibold);
  }
</style>
