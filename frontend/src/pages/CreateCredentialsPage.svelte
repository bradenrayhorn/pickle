<script lang="ts">
  import Button from "$lib/components/Button.svelte";
  import TextControl from "$lib/components/form/TextControl.svelte";
  import { getErrorQueue } from "$lib/error.svelte";
  import { CreateConnectionString, GenerateAgeKey } from "@wails/main/App";
  import { connection } from "@wails/models";

  let { onBack }: { onBack: () => void } = $props();

  const errorQueue = getErrorQueue();

  const copyButtonAction = "Copy to clipboard";
  let copyButtonText = $state(copyButtonAction);

  let url = $state("");
  let region = $state("");
  let bucket = $state("");
  let storageClass = $state("");
  let keyID = $state("");
  let keySecret = $state("");
  let ageKey = $state("");

  const isValid = $derived.by(() => {
    return [url, region, bucket, keyID, keySecret, ageKey].every(
      (value) => value.trim().length > 0,
    );
  });
</script>

<main>
  <Button
    variant="secondary"
    onclick={() => {
      onBack();
    }}>&#8592; Back</Button
  >
  <h1>Create new connection</h1>

  <form
    onsubmit={(e) => {
      e.preventDefault();

      const config = new connection.Config({
        url,
        region,
        bucket,
        storageClass,
        keyID,
        keySecret,
        ageKey,
      });
      CreateConnectionString(config)
        .then((value) => {
          navigator.clipboard.writeText(value);
          copyButtonText = "Copied!";
          setTimeout(() => {
            copyButtonText = copyButtonAction;
          }, 2500);
        })
        .catch(errorQueue.addError);
    }}
  >
    <TextControl label="Endpoint" bind:value={url} autocomplete={false} />
    <TextControl label="Region" bind:value={region} autocomplete={false} />
    <TextControl label="Bucket name" bind:value={bucket} autocomplete={false} />
    <TextControl
      label="Storage class"
      bind:value={storageClass}
      autocomplete={false}
    />
    <TextControl
      label="Access Key ID"
      bind:value={keyID}
      autocomplete={false}
    />
    <TextControl
      label="Access Key Secret"
      bind:value={keySecret}
      autocomplete={false}
    />

    <div class="age-key">
      <div class="input">
        <TextControl
          label="age encryption key"
          bind:value={ageKey}
          autocomplete={false}
        />
      </div>

      <div class="generate">
        <Button
          variant="secondary"
          type="button"
          onclick={() => {
            GenerateAgeKey()
              .then((value) => {
                ageKey = value;
              })
              .catch(errorQueue.addError);
          }}>Generate</Button
        >
      </div>
    </div>

    <Button type="submit" disabled={!isValid}>{copyButtonText}</Button>
  </form>

  <div class="note">
    <b>Note:</b> The connection info contains sensitive information and must be kept
    secret. Save connection info to a password manager.
  </div>
</main>

<style>
  main {
    padding-inline: calc(var(--spacing) * 6);
    padding-block: calc(var(--spacing) * 6);
  }

  h1 {
    font-size: var(--text-2xl);
    margin-block: calc(var(--spacing) * 4);
    font-weight: var(--font-semibold);
  }

  form {
    display: flex;
    flex-direction: column;
    gap: calc(var(--spacing) * 3);

    & .age-key {
      display: flex;
      align-items: flex-end;
      gap: var(--spacing);
      & .input {
        flex-grow: 1;
      }
      & .generate {
        flex-grow: 0;
      }
    }
  }

  .note {
    margin-top: calc(var(--spacing) * 4);
  }
</style>
