<script lang="ts">
  import FileRow from "./FileRow.svelte";
  import type { FileList } from "./files";

  let {
    fileList,
    onRefresh,
    onOpenPath,
    onDownloadFile,
  }: {
    fileList: FileList;
    onRefresh: () => void;
    onOpenPath: (path: string) => void;
    onDownloadFile: (key: string, version: string, displayName: string) => void;
  } = $props();
</script>

<ul>
  {#each fileList as file (file.type === "file" ? file.path + file.versionID : file.path)}
    <FileRow {file} {onRefresh} {onDownloadFile} {onOpenPath} />
  {/each}
</ul>

<style>
  ul {
    width: 100%;
  }
</style>
