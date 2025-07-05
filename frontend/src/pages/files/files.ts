import type { bucket } from "@wails/models";

type File = {
  type: "file";
  key: string;
  lastModified: string;
  size: string;
  versions: Array<FileVersion>;
};

type FileVersion = {
  id: string;
  lastModified: string;
  size: string;
};

type Directory = {
  type: "directory";
  key: string;
  lastModified: string;
};

export type FileList = Array<File | Directory>;

export function buildFileList(
  inDirectory: string,
  bucketFiles: Array<bucket.BucketFile>,
): FileList {
  const directories: Record<string, string> = {};
  const files: Record<string, Array<bucket.BucketFile>> = {};

  // Group into directories and files
  let filter = inDirectory;
  if (filter.startsWith("/")) {
    filter = filter.slice(1);
  }

  bucketFiles
    .filter((file) => file.name.startsWith(filter))
    .forEach((file) => {
      const parts = file.name.slice(filter.length).split("/");
      if (parts.length > 1) {
        directories[parts[0]] = file.lastModified;
      } else {
        files[file.name] = files[file.name] ?? [];
        files[file.name].push(file);
      }
    });

  // Organize files
  const fileList = Object.entries(files).map(([key, versions]) => {
    const latest = versions.find((v) => v.isLatest) ?? versions[0];

    const file: File = {
      type: "file",
      key: key.slice(filter.length),
      lastModified: latest.lastModified,
      size: latest.size,
      versions: versions.map((v) => ({
        id: v.version,
        lastModified: v.lastModified,
        size: v.size,
      })),
    };
    return file;
  });
  const directoryList: Array<Directory> = Object.entries(directories).map(
    ([key, lastModified]) => ({
      type: "directory",
      key,
      lastModified,
    }),
  );

  // Sort and return
  fileList.sort((a, b) => a.key.localeCompare(b.key));
  directoryList.sort((a, b) => a.key.localeCompare(b.key));

  return [...directoryList, ...fileList];
}
