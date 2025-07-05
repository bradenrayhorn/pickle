<script lang="ts">
  import type { FileList } from "./files";
  import IconDirectory from "~icons/mdi/folder";
  import IconFile from "~icons/mdi/file";
  import dayjs from "dayjs";

  let {
    fileList,
    onOpenDirectory,
    onDownloadFile,
  }: {
    fileList: FileList;
    onOpenDirectory: (dir: string) => void;
    onDownloadFile: (key: string) => void;
  } = $props();
</script>

<ul>
  {#each fileList as file (file.key)}
    <li>
      <button
        title={file.key}
        onclick={() => {
          if (file.type === "directory") {
            onOpenDirectory(file.key);
          } else if (file.type === "file") {
            onDownloadFile(file.key);
          }
        }}
      >
        {#if file.type === "directory"}
          <div class="icon"><IconDirectory font-size="var(--text-lg)" /></div>
          <div class="name">{file.key}</div>
          <div class="date">{dayjs(file.lastModified).format("lll")}</div>
          <div class="size"></div>
        {:else if file.type === "file"}
          <div class="icon"><IconFile font-size="var(--text-lg)" /></div>
          <div class="name">{file.key}</div>
          <div class="date">{dayjs(file.lastModified).format("lll")}</div>
          <div class="size">{file.size}</div>
        {/if}
      </button>
    </li>
  {/each}
</ul>

<style>
  ul {
    width: 100%;
  }

  li {
    width: 100%;

    &:nth-child(odd) {
      background-color: var(--color-alpha-50);
    }
    &:hover {
      background-color: var(--color-bg-primary-subtle);
    }

    transition-property: background-color;
    transition-timing-function: linear;
    transition-duration: 50ms;

    button {
      width: 100%;
      padding-inline: calc(var(--spacing) * 4);
      padding-block: calc(var(--spacing) * 2);

      cursor: pointer;

      font-size: var(--text-sm);

      display: grid;
      grid-template-columns: 1.5rem 1fr 10rem 7rem;
      align-items: center;
      gap: var(--spacing);

      .name {
        text-align: left;
        text-overflow: ellipsis;
        overflow: hidden;
      }
      .size,
      .date {
        text-align: right;
      }
    }
  }
</style>
