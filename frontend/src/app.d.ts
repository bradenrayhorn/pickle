import "unplugin-icons/types/svelte";

declare global {
  interface Window {
    __dev_pickle_forced_download_path?: string;
    __dev_pickle_forced_upload_path?: string;
  }
}
