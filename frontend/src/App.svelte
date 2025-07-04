<script lang="ts">
  import { ArchiveFile } from "@wails/App";

  let page: "connect" | "credentials" | "files" = "connect";
</script>

<main>
  <button
    onclick={() => {
      ArchiveFile();
    }}>Choose a file</button
  >

  type in ME: <input />
</main>

<style>
</style>
