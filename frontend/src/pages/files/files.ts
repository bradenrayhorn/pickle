import type { bucket } from "@wails/models";

type File = {
  type: "file";
  key: string;
  path: string;
  displayName: string;
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
  const exactFileVersions = bucketFiles.filter((f) => f.path === path);
  if (exactFileVersions.length > 0) {
    exactFileVersions.sort((a, b) =>
      b.lastModified.localeCompare(a.lastModified),
    );

    return exactFileVersions.map((file, i) => ({
      type: "file",
      key: file.key,
      path: file.path,
      displayName:
        file.path.split("/").reverse()[0] +
        ` [version ${exactFileVersions.length - i}]`,
      lastModified: file.lastModified,
      size: file.size,
      hasMultipleVersions: false,
    }));
  }

  const dirFilter = path.length > 0 ? `${path}/` : "";
  bucketFiles
    .filter((file) => file.path.startsWith(dirFilter))
    .forEach((file) => {
      const parts = file.path.slice(dirFilter.length).split("/");
      if (parts.length > 1) {
        directories.add(parts[0]);
      } else {
        files[file.path] = files[file.path] ?? [];
        files[file.path].push(file);
      }
    });

  // Organize files
  const fileList = Object.entries(files).map(([key, versions]) => {
    const latest = versions.find((v) => v.isLatest) ?? versions[0];

    const file: File = {
      type: "file",
      key: latest.key,
      path: latest.path,
      displayName: key.slice(dirFilter.length),
      lastModified: latest.lastModified,
      size: latest.size,
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

  console.log([...directoryList, ...fileList]);

  return [...directoryList, ...fileList];
}
