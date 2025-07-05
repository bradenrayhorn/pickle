<script lang="ts">
  import type { Snippet } from "svelte";
  import type { HTMLButtonAttributes } from "svelte/elements";

  type Props = {
    children: Snippet;
    variant?: "primary" | "secondary";
    icon?: boolean;
  } & HTMLButtonAttributes;

  const {
    children,
    icon = false,
    variant = "primary",
    ...rest
  }: Props = $props();
</script>

<button
  {...rest}
  part="button"
  class:icon
  class:primary={variant === "primary"}
  class:secondary={variant === "secondary"}>{@render children()}</button
>

<style>
  button {
    cursor: pointer;

    font-weight: var(--font-semibold);
    font-size: var(--text-sm);

    padding-block: calc(var(--spacing) * 0.75);
    padding-inline: calc(var(--spacing) * 3);
    border-radius: var(--radius-md);

    &.icon {
      padding: calc(var(--spacing) * 2);
    }

    &:not(:disabled) {
      &.primary {
        background-color: var(--color-bg-primary);
        color: var(--color-fg-primary);
      }
      &.secondary {
        background-color: var(--color-bg-secondary);
        color: var(--color-fg-secondary);
      }

      &:hover {
        &.primary {
          background-color: var(--color-bg-primary-hover);
        }
        &.secondary {
          background-color: var(--color-bg-secondary-hover);
        }
      }

      &:active {
        transform: translateY(calc(var(--spacing) * 0.25));
      }
    }

    &:disabled {
      background-color: var(--color-alpha-400);
      color: var(--color-alpha-400);
      cursor: not-allowed;
    }
  }
</style>
