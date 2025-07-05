<script lang="ts">
  import { initErrorQueue } from "$lib/error.svelte";
  import ErrorQueue from "$lib/ErrorQueue.svelte";
  import ConnectPage from "./pages/ConnectPage.svelte";
  import CreateCredentialsPage from "./pages/CreateCredentialsPage.svelte";
  import FilesPage from "./pages/FilesPage.svelte";

  let page: "connect" | "credentials" | "files" = $state("connect");

  initErrorQueue();
</script>

<ErrorQueue />

{#if page === "connect"}
  <ConnectPage
    onCreateCredentials={() => {
      page = "credentials";
    }}
    onConnected={() => {
      page = "files";
    }}
  />
{:else if page === "credentials"}
  <CreateCredentialsPage
    onBack={() => {
      page = "connect";
    }}
  />
{:else if page === "files"}
  <FilesPage />
{/if}
