import type { bucket } from "@wails/models";

type File = {
  type: "file";
  path: string;
  displayName: string;
  versionID: string;
  lastModified: string;
  size: string;
  hasMultipleVersions: boolean;
};

type Directory = {
  type: "directory";
  path: string;
  displayName: string;
};

export type FileList = Array<File | Directory>;
export type FileListItem = File | Directory;

export function buildFileList(
  path: string,
  bucketFiles: Array<bucket.BucketFile>,
): FileList {
  const directories = new Set<string>();
  const files: Record<string, Array<bucket.BucketFile>> = {};

  // Check if we're in "Versions" mode.
  const exactFileVersions = bucketFiles.filter((f) => f.name === path);
  if (exactFileVersions.length > 0) {
    exactFileVersions.sort((a, b) =>
      b.lastModified.localeCompare(a.lastModified),
    );

    return exactFileVersions.map((file, i) => ({
      type: "file",
      path: file.name,
      displayName: file.name + ` [version ${exactFileVersions.length - i}]`,
      lastModified: file.lastModified,
      size: file.size,
      versionID: file.version,
      hasMultipleVersions: false,
    }));
  }

  const dirFilter = path.length > 0 ? `${path}/` : "";
  bucketFiles
    .filter((file) => file.name.startsWith(dirFilter))
    .forEach((file) => {
      const parts = file.name.slice(dirFilter.length).split("/");
      if (parts.length > 1) {
        directories.add(parts[0]);
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
      path: key,
      displayName: key.slice(dirFilter.length),
      lastModified: latest.lastModified,
      size: latest.size,
      versionID: latest.version,
      hasMultipleVersions: versions.length > 1,
    };
    return file;
  });
  const directoryList: Array<Directory> = new Array(...directories).map(
    (path) => ({
      type: "directory",
      path: dirFilter + path,
      displayName: path,
    }),
  );

  // Sort and return
  fileList.sort((a, b) => a.displayName.localeCompare(b.displayName));
  directoryList.sort((a, b) => a.displayName.localeCompare(b.displayName));

  return [...directoryList, ...fileList];
}
