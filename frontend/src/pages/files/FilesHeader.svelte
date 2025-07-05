<script lang="ts">
  import Button from "$lib/components/Button.svelte";
  import UploadFile from "./UploadFile.svelte";
  import IconRefresh from "~icons/mdi/Refresh";

  let {
    prefix,
    onChangeDirectory,
    onRefresh,
  }: {
    prefix: string;
    onChangeDirectory: (dir: string) => void;
    onRefresh: () => void;
  } = $props();

  let parts = $derived(prefix.replace(/\/$/, "").split("/"));
</script>

<nav>
  <div class="path">
    <div class="breadcrumbs">
      {#each parts as part, i (part + i)}
        <button
          title={`${part}/`}
          onclick={() => {
            onChangeDirectory(parts.slice(0, i + 1).join("/"));
          }}>{`${part}/`}</button
        >
      {/each}
    </div>
  </div>

  <div class="actions">
    <Button icon variant="secondary" onclick={onRefresh}
      ><IconRefresh font-size="var(--text-lg)" /></Button
    >
    <UploadFile {onRefresh} />
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
    font-family: monospace;
    display: flex;
    align-items: center;
    overflow: hidden;
    gap: calc(var(--spacing) * 2);

    .breadcrumbs {
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
  }
</style>
