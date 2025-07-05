<script lang="ts">
  import { getErrorQueue } from "$lib/error.svelte";
  import { ListFiles } from "@wails/main/App";
  import UploadFile from "./files/UploadFile.svelte";
  import FilesHeader from "./files/FilesHeader.svelte";
  import FilesList from "./files/FilesList.svelte";
  import type { bucket } from "@wails/models";
  import { buildFileList } from "./files/files";

  const errorQueue = getErrorQueue();

  let files = $state<Array<bucket.BucketFile>>([]);
  let inDirectory = $state("/");

  const fileList = $derived.by(() => {
    return buildFileList(inDirectory, files);
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
    prefix={inDirectory}
    onChangeDirectory={(dir) => {
      inDirectory = `${dir}/`;
    }}
    onRefresh={refreshFiles}
  />
</div>

<div>
  <FilesList
    {fileList}
    onOpenDirectory={(dir) => {
      inDirectory += `${dir}/`;
    }}
  />
</div>

<style>
</style>
