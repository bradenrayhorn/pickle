<script lang="ts">
  import type { FileList } from "./files";
  import IconDirectory from "~icons/mdi/folder";
  import IconFile from "~icons/mdi/file";
  import IconFileMultiple from "~icons/mdi/file-multiple";
  import dayjs from "dayjs";

  let {
    fileList,
    onOpenPath,
    onDownloadFile,
  }: {
    fileList: FileList;
    onOpenPath: (path: string) => void;
    onDownloadFile: (key: string, version: string, displayName: string) => void;
  } = $props();
</script>

<ul>
  {#each fileList as file (file.type === "file" ? file.path + file.versionID : file.path)}
    <li>
      <button
        title={file.displayName}
        class:isDirectory={file.type === "directory"}
        onclick={() => {
          if (file.type === "directory" || file.hasMultipleVersions) {
            onOpenPath(file.path);
          } else if (file.type === "file") {
            onDownloadFile(file.path, file.versionID, file.displayName);
          }
        }}
      >
        <div class="icon">
          {#if file.type === "directory"}
            <IconDirectory
              font-size="var(--text-lg)"
              color="var(--color-bg-primary-muted)"
            />
          {:else if file.hasMultipleVersions}
            <IconFileMultiple
              font-size="var(--text-lg)"
              color="var(--color-alpha-800)"
            />
          {:else}
            <IconFile
              font-size="var(--text-lg)"
              color="var(--color-alpha-800)"
            />
          {/if}
        </div>

        <div class="name">{file.displayName}</div>
        {#if file.type === "file"}
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

      &.isDirectory {
        grid-template-columns: 1.5rem 1fr;
      }

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
