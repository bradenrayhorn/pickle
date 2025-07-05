import * as toast from "@zag-js/toast";
import { getContext, setContext } from "svelte";

const key = "toaster";

export function initToaster() {
  setContext(key, toast.createStore({ overlap: true }));
}

export function getToaster(): toast.Store {
  return getContext(key);
}

export function getErrorHandler(): (error: unknown) => void {
  const toaster = getToaster();

  return (error: unknown) => {
    let message = "An unknown error occurred";

    if (typeof error === "string") {
      message = error;
    } else if (error instanceof Error) {
      message = error.message;
    } else if (
      error &&
      typeof error === "object" &&
      "message" in error &&
      typeof error.message === "string"
    ) {
      message = error.message;
    }

    toaster.create({
      type: "error",
      title: "Error",
      description: message,
    });
  };
}
