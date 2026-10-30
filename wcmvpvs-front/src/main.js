import { createApp } from "vue";
import "./style.css";
import "./assets/admin-theme.css";
import App from "./App.vue";
import { QrcodeStream, QrcodeDropZone, QrcodeCapture } from "vue-qrcode-reader";
import { initPostHog } from "./lib/posthog";
import { resolveClubDomain } from "./clubDomain";

async function boot() {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 8000);
  try {
    const domain = await resolveClubDomain(fetch, controller.signal);
    if (domain && window.location.hostname !== domain.primaryHostname) {
      const destination = new URL(window.location.href);
      destination.protocol = "https:";
      destination.hostname = domain.primaryHostname;
      destination.port = "";
      window.location.replace(destination.href);
      return;
    }
    const app = createApp(App);
    app.provide("clubDomain", domain);
    app.component("QrcodeStream", QrcodeStream);
    app.component("QrcodeDropZone", QrcodeDropZone);
    app.component("QrcodeCapture", QrcodeCapture);
    app.mount("#app");
  } catch {
    const message = document.createElement("p");
    message.textContent =
      "Portale temporaneamente non disponibile. Riprova tra poco.";
    message.setAttribute("role", "alert");
    const retry = document.createElement("button");
    retry.type = "button";
    retry.textContent = "Riprova";
    retry.addEventListener("click", () => window.location.reload());
    document.querySelector("#app").replaceChildren(message, retry);
  } finally {
    clearTimeout(timeout);
  }
}
if (
  import.meta.env.DEV &&
  /^\/avatar-studio\/?$/.test(window.location.pathname)
) {
  import("./arena/AvatarStudio.vue").then(({ default: Studio }) =>
    createApp(Studio).mount("#app"),
  );
} else {
  initPostHog();
  boot();
}
