<script lang="ts">
  import { initToaster } from "$lib/toast/toast";
  import Toaster from "$lib/toast/Toaster.svelte";
  import ConnectPage from "./pages/ConnectPage.svelte";
  import CreateCredentialsPage from "./pages/CreateCredentialsPage.svelte";
  import FilesPage from "./pages/FilesPage.svelte";

  let page: "connect" | "credentials" | "files" = $state("connect");

  initToaster();
</script>

<Toaster />

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
