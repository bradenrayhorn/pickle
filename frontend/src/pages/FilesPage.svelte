<script lang="ts">
  import { getErrorQueue } from "$lib/error.svelte";
  import { DownloadFile, ListFiles } from "@wails/main/App";
  import type { bucket } from "@wails/models";
  import { buildFileList } from "./files/files";
  import FilesHeader from "./files/FilesHeader.svelte";
  import FilesList from "./files/FilesList.svelte";

  const errorQueue = getErrorQueue();

  let files = $state<Array<bucket.BucketFile>>([]);
  let path = $state("");

  const fileList = $derived.by(() => {
    return buildFileList(path, files);
  });

  function refreshFiles() {
    ListFiles()
      .then((res) => {
        files = res;
      })
      .catch(errorQueue.addError);
  }

  refreshFiles();
</script>

<div>
  <FilesHeader
    {path}
    onOpenPath={(newPath) => {
      path = newPath;
    }}
    onRefresh={refreshFiles}
  />
</div>

<div>
  <FilesList
    {fileList}
    onOpenPath={(newPath) => {
      path = newPath;
    }}
    onDownloadFile={(key, version) => {
      DownloadFile(key, version).catch(errorQueue.addError);
    }}
  />
</div>

<style>
</style>
