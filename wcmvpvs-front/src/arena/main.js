import { createApp } from "vue";
import ArenaApp from "./ArenaApp.vue";
import OrganizationArena from "./OrganizationArena.vue";
import "./arena.css";
import "./fans.css";
import "./immersive.css";

createApp(
  new URLSearchParams(window.location.search).has("classic")
    ? ArenaApp
    : OrganizationArena,
  { slug: "arena-demo", demo: true },
).mount("#arena-app");
