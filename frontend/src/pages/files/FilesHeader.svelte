<script lang="ts">
  import Button from "$lib/components/Button.svelte";
  import UploadFile from "./UploadFile.svelte";
  import IconRefresh from "~icons/mdi/Refresh";
  import IconTrashBin from "~icons/mdi/trash-can-outline";
  import IconTrashBinExit from "~icons/mdi/exit-run";

  let {
    path,
    isInTrashBin,
    onOpenTrashBin,
    onCloseTrashBin,
    onOpenPath,
    onRefresh,
  }: {
    path: string;
    isInTrashBin: boolean;
    onOpenTrashBin: () => void;
    onCloseTrashBin: () => void;
    onOpenPath: (path: string) => void;
    onRefresh: () => void;
  } = $props();

  let parts = $derived(path.length > 0 ? ["", ...path.split("/")] : [""]);
</script>

<nav>
  <div class="path">
    <div class="breadcrumbs">
      {#each parts as part, i (part + i)}
        <button
          title={`${part}/`}
          onclick={() => {
            onOpenPath(parts.slice(1, i + 1).join("/"));
          }}
        >
          {`${part}/`}
        </button>
      {/each}
    </div>
  </div>

  <div class="actions">
    {#if isInTrashBin}
      <Button icon onclick={onCloseTrashBin}>
        <div class="exit-trash">
          <div>Exit Trash Bin</div>
          <IconTrashBinExit font-size="var(--text-lg)" />
        </div>
      </Button>
    {:else}
      <Button icon variant="secondary" onclick={onOpenTrashBin}>
        <IconTrashBin font-size="var(--text-lg)" />
      </Button>

      <Button icon variant="secondary" onclick={onRefresh}>
        <IconRefresh font-size="var(--text-lg)" />
      </Button>

      <UploadFile {onRefresh} {path} />
    {/if}
  </div>
</nav>

<style>
  nav {
    display: flex;
    justify-content: space-between;
    align-items: center;

    width: 100%;
    padding-block: calc(var(--spacing) * 2);
    padding-inline: calc(var(--spacing) * 4);
  }

  .path {
    display: flex;
    align-items: center;
    overflow: hidden;
    gap: calc(var(--spacing) * 2);

    .breadcrumbs {
      font-family: monospace;
      display: flex;
      align-items: center;
      flex-wrap: wrap;
      gap: calc(var(--spacing) * 1);
      overflow: hidden;

      button {
        cursor: pointer;
        padding: calc(var(--spacing));

        text-overflow: ellipsis;
        overflow: hidden;

        &:hover {
          text-decoration: underline;
        }
      }
    }
  }

  .actions {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: calc(var(--spacing) * 2);

    & .exit-trash {
      display: flex;
      align-items: center;
      gap: calc(var(--spacing) * 1);
    }
  }
</style>
