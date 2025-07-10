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

<ul>
  {#each fileList as file (file.type === "file" ? file.key : file.path)}
    <FileRow {file} {isInTrashBin} {onRefresh} {onDownloadFile} {onOpenPath} />
  {/each}
</ul>

<style>
  ul {
    width: 100%;
  }
</style>
