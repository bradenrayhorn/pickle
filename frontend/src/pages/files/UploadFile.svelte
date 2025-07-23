<script lang="ts">
  import Button from "$lib/components/Button.svelte";
  import Input from "$lib/components/Input.svelte";
  import { getErrorHandler, getToaster } from "$lib/toast/toast";
  import { SelectFile, UploadFile } from "@wails/main/App";
  import IconUpload from "~icons/mdi/FileUploadOutline";

  let {
    onRefresh,
    path: currentPath,
  }: { onRefresh: () => void; path: string } = $props();

  const toaster = getToaster();
  const onError = getErrorHandler();

  let uploadDialog: HTMLDialogElement | undefined = undefined;

  let pendingFilePath = $state("");
  let pendingFileName = $state("");
</script>

<Button
  icon
  onclick={() => {
    const forcedUploadPath = window.__dev_pickle_forced_upload_path ?? ""; // only for e2e tests

    (forcedUploadPath ? Promise.resolve(forcedUploadPath) : SelectFile())
      .then((path) => {
        if (path !== "") {
          pendingFilePath = path;

          const prefix = currentPath.length > 1 ? `${currentPath}/` : "";
          pendingFileName = prefix + path.replace(/^.*[\\/]/, "");

          uploadDialog?.showModal();
        }
      })
      .catch(onError);
  }}
>
  <div class="upload-button">
    <div>Upload</div>
    <IconUpload font-size="var(--text-lg)" />
  </div>
</Button>

<dialog bind:this={uploadDialog}>
  <h1>Upload file</h1>

  <h2>Enter the new name of the file, including its directory.</h2>

  <div class="entry">
    <Input
      bind:value={pendingFileName}
      autocomplete="off"
      autocorrect="off"
      autocapitalize="off"
    />
    <span
      >Use <code>/</code> to create directories. For example
      <code>taxes/2024.zip</code>
      will create a <code>taxes</code> folder (if it doesn't exist) and with the
      <code>2024.zip</code> file inside.</span
    >
  </div>

  <div class="actions">
    <Button
      variant="secondary"
      onclick={() => {
        uploadDialog?.close();
      }}
    >
      Cancel
    </Button>
    <Button
      onclick={() => {
        uploadDialog?.close();

        const toastID = toaster.create({
          type: "loading",
          title: pendingFileName,
          description: "Uploading file...",
        });
        UploadFile(pendingFilePath, pendingFileName)
          .then(() => {
            pendingFilePath = "";
            pendingFileName = "";
            onRefresh();
            toaster.update(toastID, {
              type: "success",
              description: "File successfully uploaded!",
              duration: 2500,
            });
          })
          .catch((error) => {
            toaster.remove(toastID);
            onError(error);
          });
      }}>Upload</Button
    >
  </div>
</dialog>

<style>
  .upload-button {
    display: flex;
    align-items: center;
    gap: calc(var(--spacing) * 1);
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

    h1 {
      font-size: var(--text-lg);
      font-weight: var(--font-semibold);
      margin-bottom: calc(var(--spacing) * 4);
    }

    h2 {
      font-size: var(--text-sm);
      font-weight: var(--font-semibold);
      margin-bottom: calc(var(--spacing) * 3);
    }

    .entry {
      display: grid;
      span {
        margin-top: calc(var(--spacing) * 2);
        font-size: var(--text-sm);
      }
    }

    .actions {
      display: flex;
      align-items: center;
      justify-content: flex-end;
      gap: calc(var(--spacing) * 2);

      margin-top: calc(var(--spacing) * 8);
    }

    &::backdrop {
      background: var(--color-modal-backdrop);
    }
  }
</style>
