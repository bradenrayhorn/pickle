<script lang="ts">
  import Input from "../Input.svelte";
  import type { HTMLInputAttributes } from "svelte/elements";

  type Props = {
    label: string;
    value: string;
    autocomplete?: boolean;
    inputProps?: Omit<HTMLInputAttributes, "value" | "autocomplete">;
  };

  let {
    label,
    value = $bindable(""),
    autocomplete = true,
    inputProps,
  }: Props = $props();
</script>

<label>
  <span>{label}</span>

  <Input
    {...inputProps}
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
