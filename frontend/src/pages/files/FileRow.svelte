<script lang="ts">
  import type { FileListItem } from "./files";
  import IconDirectory from "~icons/mdi/folder";
  import IconFile from "~icons/mdi/file";
  import IconFileMultiple from "~icons/mdi/file-multiple";
  import dayjs from "dayjs";
  import Button from "$lib/components/Button.svelte";
  import { DeleteFile, RestoreFile } from "@wails/main/App";
  import { getErrorHandler, getToaster } from "$lib/toast/toast";

  let {
    file,
    isInTrashBin,
    onRefresh,
    onOpenPath,
    onDownloadFile,
  }: {
    file: FileListItem;
    isInTrashBin: boolean;
    onRefresh: () => void;
    onOpenPath: (path: string) => void;
    onDownloadFile: (path: string, displayName: string) => void;
  } = $props();

  let isDeleting = $state(false);

  const toaster = getToaster();
  const onError = getErrorHandler();

  let actionsDialog: HTMLDialogElement | undefined = $state(undefined);
</script>

<li>
  <button
    title={file.displayName}
    class:isDirectory={file.type === "directory"}
    onclick={() => {
      if (file.type === "directory" || file.hasMultipleVersions) {
        onOpenPath(file.path);
      } else if (file.type === "file") {
        actionsDialog?.showModal();
      }
    }}
  >
    <div class="icon">
      {#if file.type === "directory"}
        <IconDirectory
          font-size="var(--text-lg)"
          color="var(--color-bg-primary-muted)"
        />
      {:else if file.hasMultipleVersions}
        <IconFileMultiple
          font-size="var(--text-lg)"
          color="var(--color-alpha-800)"
        />
      {:else}
        <IconFile font-size="var(--text-lg)" color="var(--color-alpha-800)" />
      {/if}
    </div>

    <div class="name">{file.displayName}</div>
    {#if file.type === "file"}
      <div class="date">{dayjs(file.lastModified).format("lll")}</div>
      <div class="size">{file.size}</div>
    {/if}
  </button>
</li>

{#if file.type === "file"}
  <dialog bind:this={actionsDialog} closedby={isDeleting ? "none" : "any"}>
    <h2>{file.displayName}</h2>

    <div class="actions">
      <Button
        variant="secondary"
        disabled={isDeleting}
        onclick={() => {
          actionsDialog?.close();
        }}
      >
        Cancel
      </Button>

      {#if !isInTrashBin}
        <Button
          isLoading={isDeleting}
          onclick={() => {
            isDeleting = true;
            DeleteFile(file.key)
              .then(() => {
                toaster.create({
                  type: "success",
                  title: file.displayName,
                  description: "Successfully deleted!",
                });
                actionsDialog?.close();
                onRefresh();
              })
              .catch(onError)
              .finally(() => {
                isDeleting = false;
              });
          }}
          variant="destructive">Delete</Button
        >
      {:else}
        <Button
          isLoading={isDeleting}
          onclick={() => {
            isDeleting = true;
            RestoreFile(file.key)
              .then(() => {
                toaster.create({
                  type: "success",
                  title: file.displayName,
                  description: "Successfully restored!",
                });
                actionsDialog?.close();
                onRefresh();
              })
              .catch(onError)
              .finally(() => {
                isDeleting = false;
              });
          }}
          variant="destructive">Restore</Button
        >
      {/if}

      <Button
        disabled={isDeleting}
        onclick={() => {
          onDownloadFile(file.key, file.displayName);
          actionsDialog?.close();
        }}>Download</Button
      >
    </div>
  </dialog>
{/if}

<style>
  li {
    width: 100%;

    &:nth-child(odd) {
      background-color: var(--color-alpha-50);
    }
    &:hover {
      background-color: var(--color-bg-primary-subtle);
    }

    transition-property: background-color;
    transition-timing-function: linear;
    transition-duration: 50ms;

    button {
      width: 100%;
      padding-inline: calc(var(--spacing) * 4);
      padding-block: calc(var(--spacing) * 2);

      cursor: pointer;

      font-size: var(--text-sm);

      display: grid;
      grid-template-columns: 1.5rem 1fr 10rem 7rem;
      align-items: center;
      gap: var(--spacing);

      &.isDirectory {
        grid-template-columns: 1.5rem 1fr;
      }

      .name {
        text-align: left;
        text-overflow: ellipsis;
        overflow: hidden;
      }
      .size,
      .date {
        text-align: right;
      }
    }
  }

  dialog {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    background: var(--color-bg-elevated);
    color: var(--color-fg-elevated);
    padding: calc(var(--spacing) * 4);

    border-color: var(--color-alpha-200);
    border-bottom-width: calc(var(--spacing) * 1);
    border-left-width: calc(var(--spacing) * 1);

    min-width: calc(var(--spacing) * 90);

    h2 {
      font-size: var(--text-md);
      font-weight: var(--font-semibold);
      margin-bottom: calc(var(--spacing) * 6);
    }

    .actions {
      display: flex;
      flex-direction: row;
      gap: calc(var(--spacing) * 4);

      margin-top: calc(var(--spacing) * 8);
    }

    &::backdrop {
      background: var(--color-modal-backdrop);
    }
  }
</style>
