<script lang="ts">
  import Button from "$lib/components/Button.svelte";
  import TextControl from "$lib/components/form/TextControl.svelte";
  import { getErrorQueue } from "$lib/error.svelte";
  import { SelectFile, UploadFile } from "@wails/main/App";

  let { onRefresh }: { onRefresh: () => void } = $props();

  const errorQueue = getErrorQueue();

  let pendingFile = $state("");
  let pendingFileName = $state("");
</script>

<Button
  onclick={() => {
    SelectFile()
      .then((path) => {
        pendingFile = path;

        const fileName = path.replace(/^.*[\\\/]/, "");
        pendingFileName = fileName;
      })
      .catch(errorQueue.addError);
  }}>Upload file</Button
>

{#if pendingFile}
  <dialog open>
    Confirm file will be uploaded.

    <TextControl label="target directory" bind:value={pendingFileName} />

    <Button
      onclick={() => {
        UploadFile(pendingFile, pendingFileName)
          .then(() => {
            pendingFile = "";
            pendingFileName = "";
            onRefresh();
          })
          .catch(errorQueue.addError);
      }}>Upload my file</Button
    >
  </dialog>
{/if}

<style>
</style>
