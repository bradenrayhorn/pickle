<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";

  type Props = HTMLInputAttributes;

  let { children, value = $bindable(), ...rest }: Props = $props();
</script>

<input bind:value {...rest} />

<style>
  input {
    border: 1px solid var(--color-alpha-400);
    border-radius: var(--radius-md);

    padding-inline: calc(var(--spacing) * 2);
    padding-block: calc(var(--spacing) * 1);

    font-size: var(--text-sm);

    &:focus {
      outline: none;
    }
  }
</style>
