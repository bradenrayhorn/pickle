<script lang="ts">
  import Button from "$lib/components/Button.svelte";
  import TextControl from "$lib/components/form/TextControl.svelte";
  import { getErrorHandler } from "$lib/toast/toast";
  import { InitializeConnection } from "@wails/main/App";

  type Props = {
    onConnected: () => void;
    onCreateCredentials: () => void;
  };

  const { onConnected, onCreateCredentials }: Props = $props();

  const onError = getErrorHandler();

  let credentials = $state("");
</script>

<div class="wrapper">
  <h1>pickle</h1>

  <div class="form">
    <TextControl
      bind:value={credentials}
      label="Connection credentials"
      inputProps={{
        type: "password",
        placeholder: "Paste connection info here",
      }}
    />

    <Button
      onclick={() => {
        InitializeConnection(credentials)
          .then(() => {
            onConnected();
          })
          .catch(onError);
      }}
    >
      Connect
    </Button>
  </div>

  <div class="subaction">
    <Button
      variant="secondary"
      onclick={() => {
        onCreateCredentials();
      }}
    >
      Create new connection
    </Button>
  </div>
</div>

<style>
  .wrapper {
    display: flex;
    flex-direction: column;
    padding-inline: calc(var(--spacing) * 8);
  }

  .form {
    display: flex;
    flex-direction: column;

    align-items: center;

    gap: calc(var(--spacing) * 4);
  }

  .subaction {
    margin-top: calc(var(--spacing) * 6);
    text-align: right;
  }

  h1 {
    text-align: center;
    font-size: var(--text-2xl);
    font-weight: var(--font-bold);

    margin-bottom: calc(var(--spacing) * 4);
  }
</style>
