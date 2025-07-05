<script lang="ts">
  import Input from "../Input.svelte";
  import type { HTMLInputAttributes } from "svelte/elements";

  type Props = {
    label: string;
    value: string;
    autocomplete?: boolean;
  };

  let { label, value = $bindable(""), autocomplete = true }: Props = $props();
</script>

<label>
  <span>{label}</span>

  <Input
    bind:value
    autocomplete={autocomplete ? "on" : "off"}
    autocorrect={autocomplete ? "on" : "off"}
    autocapitalize={autocomplete ? "on" : "off"}
    spellcheck={autocomplete}
  />
</label>

<style>
  label {
    display: grid;

    & > span {
      font-size: var(--text-sm);
      margin-block-end: var(--spacing);
    }
  }
</style>
