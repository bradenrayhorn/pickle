<script lang="ts">
  import FileRow from "./FileRow.svelte";
  import type { FileList } from "./files";

  let {
    fileList,
    isInTrashBin,
    onRefresh,
    onOpenPath,
    onDownloadFile,
  }: {
    fileList: FileList;
    isInTrashBin: boolean;
    onRefresh: () => void;
    onOpenPath: (path: string) => void;
    onDownloadFile: (key: string, displayName: string) => void;
  } = $props();
</script>

<table>
  <thead>
    <tr>
      <th scope="col" class="hidden">Type</th>
      <th scope="col">Name</th>
      <th scope="col" class="right">Date uploaded</th>
      <th scope="col" class="right">Size</th>
    </tr>
  </thead>
  <tbody>
    {#each fileList as file (file.type === "file" ? file.key : file.path)}
      <FileRow
        {file}
        {isInTrashBin}
        {onRefresh}
        {onDownloadFile}
        {onOpenPath}
      />
    {/each}
  </tbody>
</table>

<style>
  table {
    width: 100%;
  }

  thead {
    & tr {
      display: grid;
      grid-template-columns: 1.5rem 1fr 10rem 7rem;
      align-items: center;
      gap: var(--spacing);
      padding-inline: calc(var(--spacing) * 4);
      padding-block: calc(var(--spacing) * 2);

      & th {
        text-align: left;
        font-size: var(--text-sm);
        &.hidden {
          overflow: hidden;
          visibility: hidden;
        }
        &.right {
          text-align: right;
        }
      }
    }
  }
</style>
