<script lang="ts">
  import { DownloadFile, ListFiles } from "@wails/main/App";
  import type { bucket } from "@wails/models";
  import { buildFileList } from "./files/files";
  import FilesHeader from "./files/FilesHeader.svelte";
  import FilesList from "./files/FilesList.svelte";
  import { getErrorHandler, getToaster } from "$lib/toast/toast";
  import { EventsOn } from "@wails-runtime/runtime";
  import { onDestroy } from "svelte";

  const toaster = getToaster();
  const onError = getErrorHandler();

  let files = $state<Array<bucket.BucketFile>>([]);
  let path = $state("");

  const fileList = $derived.by(() => {
    return buildFileList(path, files);
  });

  // download progress
  const downloadNameMap: Record<string, string> = {};
  const unregister = [
    EventsOn("download-start", (downloadID: string) => {
      toaster.create({
        id: downloadID,
        type: "loading",
        title: downloadNameMap[downloadID] ?? "?? File ??",
        description: `Downloading...`,
      });
    }),
    EventsOn("download-complete", (downloadID: string) => {
      toaster.update(downloadID, {
        type: "success",
        title: downloadNameMap[downloadID] ?? "?? File ??",
        description: `Download complete!`,
      });
    }),
  ];
  onDestroy(() => {
    unregister.forEach((rm) => rm());
  });

  // file list
  function refreshFiles() {
    ListFiles()
      .then((res) => {
        files = res;
      })
      .catch(onError);
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
    onDownloadFile={(key, version, displayName) => {
      const bytes = new Uint8Array(16);
      crypto.getRandomValues(bytes);
      const downloadID = btoa(String.fromCharCode(...bytes));

      downloadNameMap[downloadID] = displayName;
      DownloadFile(key, version, downloadID)
        .catch((error) => {
          toaster.remove(downloadID);
          onError(error);
        })
        .finally(() => {
          delete downloadNameMap[downloadID];
        });
    }}
  />
</div>

<style>
</style>
