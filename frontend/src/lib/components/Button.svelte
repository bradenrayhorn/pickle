<script lang="ts">
  import type { Snippet } from "svelte";
  import type { HTMLButtonAttributes } from "svelte/elements";
  import IconLoading from "~icons/mdi/loading";

  type Props = {
    children: Snippet;
    variant?: "primary" | "secondary" | "destructive";
    isLoading?: boolean;
    icon?: boolean;
  } & HTMLButtonAttributes;

  const {
    children,
    icon = false,
    isLoading = false,
    variant = "primary",
    ...rest
  }: Props = $props();
</script>

<button
  {...rest}
  disabled={isLoading || (rest.disabled ?? false)}
  class:icon
  class:isLoading
  class:primary={variant === "primary"}
  class:destructive={variant === "destructive"}
  class:secondary={variant === "secondary"}
>
  {#if isLoading}
    <div class="loading">
      <span>{@render children()}</span>
      <div class="spinner"><IconLoading /></div>
    </div>
  {:else}
    {@render children()}
  {/if}
</button>

<style>
  button {
    cursor: pointer;

    font-weight: var(--font-semibold);
    font-size: var(--text-sm);

    padding-block: calc(var(--spacing) * 0.75);
    padding-inline: calc(var(--spacing) * 3);
    border-radius: var(--radius-md);

    & .loading {
      position: relative;
      & span {
        visibility: hidden;
      }
      & .spinner {
        position: absolute;
        top: 50%;
        left: 50%;
        animation: spin 1s linear infinite;
      }
    }

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
      &.destructive {
        background-color: var(--color-bg-error);
        color: var(--color-fg-error);
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

  @keyframes spin {
    0% {
      transform: translate(-50%, -50%) rotate(0deg);
    }
    100% {
      transform: translate(-50%, -50%) rotate(360deg);
    }
  }
</style>
