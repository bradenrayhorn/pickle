<script lang="ts">
  import { normalizeProps, useMachine } from "@zag-js/svelte";
  import * as toast from "@zag-js/toast";
  import IconClose from "~icons/mdi/close";

  type Props = {
    toast: toast.Options;
    index: number;
    parent: toast.GroupService;
  };

  const { toast: options, index, parent }: Props = $props();

  const machineProps = $derived({ ...options, parent, index });
  const service = useMachine(toast.machine, () => machineProps);

  const api = $derived(toast.connect(service, normalizeProps));
</script>

<div {...api.getRootProps()}>
  <div class="content">
    {#if api.title}
      <h3 {...api.getTitleProps()}>{api.title}</h3>
    {/if}

    {#if api.description}
      <p {...api.getDescriptionProps()}>{api.description}</p>
    {/if}
  </div>

  <div class="close">
    <button onclick={api.dismiss} aria-label="Close"><IconClose /></button>
  </div>
</div>

<style>
  [data-part="root"] {
    translate: var(--x) var(--y);
    scale: var(--scale);
    z-index: var(--z-index);
    height: var(--height);
    opacity: var(--opacity);
    will-change: translate, opacity, scale;

    min-width: calc(var(--spacing) * 60);
    max-width: calc(var(--spacing) * 200);
    overflow: hidden;

    transition:
      translate 400ms,
      scale 400ms,
      opacity 400ms;
    transition-timing-function: cubic-bezier(0.21, 1.02, 0.73, 1);

    &[data-state="closed"] {
      transition:
        translate 400ms,
        scale 400ms,
        opacity 200ms;
      transition-timing-function: cubic-bezier(0.06, 0.71, 0.55, 1);
    }

    border-top-right-radius: var(--radius-md);

    display: flex;

    & .content {
      flex-grow: 1;
      padding-inline-start: calc(var(--spacing) * 4);
      padding-block: calc(var(--spacing) * 2);
    }
    & .close {
      flex-shrink: 0;
      align-self: flex-start;

      & button {
        cursor: pointer;

        margin-block: calc(var(--spacing) * 1);
        margin-inline-end: calc(var(--spacing) * 1);
        padding: calc(var(--spacing) * 0.25);

        &:hover {
          background-color: var(--color-alpha-200);
        }
      }
    }

    background-color: var(--color-bg-elevated);
    color: var(--color-fg-elevated);

    border-top-color: var(--color-alpha-50);
    border-right-color: var(--color-alpha-50);
    border-top-width: var(--spacing) * 0.5;
    border-right-width: var(--spacing) * 0.5;

    border-bottom-width: var(--spacing);
    border-left-width: var(--spacing);

    &[data-type="error"] {
      border-bottom-color: var(--color-accent-error);
      border-left-color: var(--color-accent-error);
    }
    &[data-type="loading"] {
      border-bottom-color: var(--color-accent-loading);
      border-left-color: var(--color-accent-loading);
    }
    &[data-type="success"] {
      border-bottom-color: var(--color-accent-success);
      border-left-color: var(--color-accent-success);
    }
  }

  [data-part="title"] {
    font-weight: var(--font-bold);
  }

  [data-part="description"] {
    font-weight: var(--font-medium);
    font-size: var(--text-sm);
  }
</style>
