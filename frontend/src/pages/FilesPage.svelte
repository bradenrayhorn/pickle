<script lang="ts">
  import { DownloadFile, ListFiles, ListFilesInTrash } from "@wails/main/App";
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
  let isInTrashBin = $state(false);

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

      delete downloadNameMap[downloadID];
    }),
  ];
  onDestroy(() => {
    unregister.forEach((rm) => rm());
  });

  // file list
  function refreshFiles() {
    (isInTrashBin ? ListFilesInTrash() : ListFiles())
      .then((res) => {
        files = res;
        if (path !== "" && !files.some((f) => f.path.startsWith(`${path}/`))) {
          path = "";
        }
      })
      .catch(onError);
  }

  refreshFiles();
</script>

<div>
  <FilesHeader
    {path}
    {isInTrashBin}
    onOpenTrashBin={() => {
      isInTrashBin = true;
      refreshFiles();
    }}
    onCloseTrashBin={() => {
      isInTrashBin = false;
      refreshFiles();
    }}
    onOpenPath={(newPath) => {
      path = newPath;
    }}
    onRefresh={refreshFiles}
  />
</div>

<div>
  <FilesList
    {fileList}
    {isInTrashBin}
    onRefresh={refreshFiles}
    onOpenPath={(newPath) => {
      path = newPath;
    }}
    onDownloadFile={(key, displayName) => {
      const bytes = new Uint8Array(16);
      crypto.getRandomValues(bytes);
      const downloadID = btoa(String.fromCharCode(...bytes));

      downloadNameMap[downloadID] = displayName;
      DownloadFile(
        key,
        downloadID,
        window.__dev_pickle_forced_download_path ?? "", // only for e2e tests
      ).catch((error) => {
        toaster.remove(downloadID);
        onError(error);
        delete downloadNameMap[downloadID];
      });
    }}
  />
</div>

<style>
</style>
