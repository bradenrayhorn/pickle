import { getContext, setContext } from "svelte";

class ErrorQueue {
  errors = $state<Array<string>>([]);

  addError = (error: unknown) => {
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

    this.errors.push(message);
  };
}

const key = "error-queue";

export function initErrorQueue() {
  setContext(key, new ErrorQueue());
}

export function getErrorQueue(): ErrorQueue {
  return getContext(key);
}
