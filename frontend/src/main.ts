import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";
import localizedFormat from "dayjs/plugin/localizedFormat";
import dayjs from "dayjs";

dayjs.extend(localizedFormat);

const app = mount(App, {
  target: document.getElementById("app")!,
});

export default app;
