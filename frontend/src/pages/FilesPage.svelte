<script lang="ts">
  import { getErrorQueue } from "$lib/error.svelte";
  import { ListFiles } from "@wails/main/App";
  import UploadFile from "./files/UploadFile.svelte";

  const errorQueue = getErrorQueue();

  let files = $state<Array<string>>([]);

  function refreshFiles() {
    ListFiles()
      .then((file) => {
        files = file.map((f) => JSON.stringify(f));
      })
      .catch(errorQueue.addError);
  }

  refreshFiles();
</script>

<div>
  Header info
  <UploadFile onRefresh={() => refreshFiles()} />
</div>

<div>
  files:

  {#each files as file (file)}
    <div>{file}</div>
  {/each}
</div>

<style>
</style>
