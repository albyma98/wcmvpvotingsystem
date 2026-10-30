<script setup>
import { ref, onMounted, onBeforeUnmount, watch, computed } from "vue";
import * as THREE from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import { createCrowd } from "./crowd/createCrowd.js";
import { createDemoSession } from "./actions/demoSession.js";
import { createStands } from "./scene/stands.js";
import AppearanceControls from "./crowd/AppearanceControls.vue";
const canvas = ref(null),
  equipment = ref({}),
  lod = ref("auto"),
  standing = ref(false),
  population = ref(false),
  metrics = ref({ fps: 0, calls: 0, triangles: 0, detailed: 0, poseMs: 0 }),
  error = ref("");
try {
  equipment.value = JSON.parse(
    localStorage.getItem("abx.avatar-studio.v1") || "{}",
  );
} catch {}
let scene,
  renderer,
  camera,
  controls,
  crowd,
  resizeObserver,
  actor,
  seat,
  stands,
  hidden = document.hidden,
  lastStats = 0,
  frames = 0;
function disposeGroup(root) {
  const gs = new Set(),
    ms = new Set();
  root.traverse((o) => {
    if (o.geometry) gs.add(o.geometry);
    if (o.material) ms.add(o.material);
    if (o.isInstancedMesh) o.dispose();
  });
  gs.forEach((g) => g.dispose());
  ms.forEach((m) => m.dispose());
  root.removeFromParent();
}
function build() {
  if (crowd) disposeGroup(crowd.group);
  let actors;
  if (population.value) {
    actors = createDemoSession().actors;
    actor = actors[0];
    actor.equipment = { ...equipment.value };
    camera.position.set(17, 11, -23);
    controls.target.set(0, 3, 0);
  } else {
    actor = {
      id: "studio",
      kind: "profile",
      seed: 101,
      equipment: { ...equipment.value },
      profile: { nickname: "Studio", active: false },
      seat: {
        id: "studio-seat",
        x: 0,
        y: 0.31,
        z: 0,
        yaw: 0,
        side: 1,
        stand: "B",
        row: 1,
        column: 20,
        hasSeat: true,
      },
      poseOverride: { standing: standing.value ? 1 : 0 },
    };
    actors = [actor];
    camera.position.set(1.1, 1.2, -2.5);
    controls.target.set(0, 0.76, 0);
  }
  seat.visible = !population.value;
  stands.visible = population.value;
  crowd = createCrowd(scene, actors, {
    selectedId: actor.id,
    lod: lod.value,
    markers: false,
  });
  controls.update();
}
function appearance(values) {
  const next = { ...equipment.value };
  for (const [key, value] of Object.entries(values))
    next["look-" + key] = String(value);
  if ("hair" in values) delete next.hair;
  if ("shirt" in values) delete next.shirt;
  equipment.value = next;
}
function toggle(key, value) {
  const next = { ...equipment.value };
  if (value === "") delete next[key];
  else next[key] = value;
  equipment.value = next;
}
function animate(kind) {
  crowd?.trigger(kind, actor.id);
}
function visibility() {
  hidden = document.hidden;
}
watch(
  equipment,
  (value) => {
    try {
      localStorage.setItem("abx.avatar-studio.v1", JSON.stringify(value));
    } catch {}
    crowd?.setEquipment(actor.id, value);
  },
  { deep: true },
);
watch(population, () => {
  if (renderer) build();
});
watch(lod, (value) => crowd?.setLod(value));
watch(standing, (value) => {
  if (actor && !population.value)
    actor.poseOverride = { standing: value ? 1 : 0 };
  if (controls && !population.value) {
    camera.position.set(1.1, value ? 1.4 : 1.2, value ? -2.8 : -2.5);
    controls.target.set(0, value ? 0.82 : 0.76, 0);
  }
});
onMounted(() => {
  document.title = "Avatar Studio · Arena Tifosi";
  try {
    scene = new THREE.Scene();
    scene.background = new THREE.Color("#e8e9df");
    camera = new THREE.PerspectiveCamera(38, 1, 0.05, 160);
    renderer = new THREE.WebGLRenderer({
      antialias: true,
      powerPreference: "high-performance",
    });
    renderer.outputColorSpace = THREE.SRGBColorSpace;
    renderer.toneMapping = THREE.ACESFilmicToneMapping;
    renderer.toneMappingExposure = 1.08;
    renderer.domElement.setAttribute(
      "aria-label",
      "Anteprima avatar 3D: trascina per ruotare a 360 gradi, usa la rotella o il pinch per avvicinarti.",
    );
    renderer.domElement.setAttribute("role", "img");
    canvas.value.appendChild(renderer.domElement);
    scene.add(new THREE.HemisphereLight("#fff9ec", "#7d897b", 2));
    const key = new THREE.DirectionalLight("#fff1dc", 2.7);
    key.position.set(-3, 6, -5);
    scene.add(key);
    const rim = new THREE.DirectionalLight("#d2e3f2", 1.5);
    rim.position.set(4, 4, 3);
    scene.add(rim);
    const floor = new THREE.Mesh(
      new THREE.CircleGeometry(38, 64),
      new THREE.MeshStandardMaterial({ color: "#dde0d2", roughness: 1 }),
    );
    floor.rotation.x = -Math.PI / 2;
    floor.position.y = -0.015;
    scene.add(floor);
    seat = new THREE.Group();
    scene.add(seat);
    for (const [size, position, color] of [
      [[0.62, 0.31, 0.38], [0, 0.155, 0], "#c9cdbb"],
      [[0.47, 0.1, 0.29], [0, 0.41, 0], "#355391"],
      [[0.47, 0.34, 0.065], [0, 0.59, 0.14], "#355391"],
    ]) {
      const mesh = new THREE.Mesh(
        new THREE.BoxGeometry(...size),
        new THREE.MeshStandardMaterial({ color, roughness: 0.8 }),
      );
      mesh.position.set(...position);
      seat.add(mesh);
    }
    stands = new THREE.Group();
    createStands(stands);
    scene.add(stands);
    controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.minDistance = 1.1;
    controls.maxDistance = 65;
    controls.maxPolarAngle = Math.PI / 2 + 0.08;
    build();
    const resize = () => {
      const r = canvas.value.getBoundingClientRect();
      camera.aspect = r.width / r.height;
      camera.updateProjectionMatrix();
      renderer.setPixelRatio(
        Math.min(
          devicePixelRatio || 1,
          innerWidth < 700 ? 1.25 : 1.5,
          Math.sqrt(2200000 / (r.width * r.height)),
        ),
      );
      renderer.setSize(r.width, r.height, false);
    };
    resizeObserver = new ResizeObserver(resize);
    resizeObserver.observe(canvas.value);
    resize();
    document.addEventListener("visibilitychange", visibility);
    lastStats = performance.now();
    renderer.setAnimationLoop((now) => {
      if (hidden) return;
      controls.update();
      crowd.update(now, camera);
      renderer.render(scene, camera);
      frames++;
      if (now - lastStats > 1000) {
        const stats = crowd.stats();
        metrics.value = {
          fps: Math.round((frames * 1000) / (now - lastStats)),
          calls: renderer.info.render.calls,
          triangles: renderer.info.render.triangles,
          detailed: stats.detailed,
          poseMs: stats.poseMs.toFixed(2),
        };
        lastStats = now;
        frames = 0;
      }
    });
  } catch (e) {
    error.value = e.message;
  }
});
onBeforeUnmount(() => {
  renderer?.setAnimationLoop(null);
  controls?.dispose();
  resizeObserver?.disconnect();
  document.removeEventListener("visibilitychange", visibility);
  if (scene) disposeGroup(scene);
  renderer?.dispose();
  renderer?.domElement.remove();
});
</script>
<template>
  <main class="avatar-studio">
    <section ref="canvas" class="studio-canvas"></section>
    <header>
      <small>ARENA TIFOSI · LABORATORIO DI SVILUPPO</small>
      <h1>Avatar Studio</h1>
      <p>Cartoon heroic · geometrie condivise · stesso rig degli spalti</p>
      <a href="/arena-demo/">← Torna all’arena</a>
    </header>
    <aside>
      <h2>Il personaggio</h2>
      <AppearanceControls
        :equipment="equipment"
        :seed="101"
        @change="appearance"
      /><label
        >Maglia del guardaroba<select
          :value="equipment.shirt || ''"
          @change="toggle('shirt', $event.target.value)"
        >
          <option value="">Maglia base</option>
          <option value="club-shirt">Maglia della società</option>
          <option value="away-shirt">Maglia alternativa</option>
          <option value="hoodie">Felpa</option>
          <option value="shirt-color">Verde lime</option>
        </select></label
      >
      <div class="studio-accessories">
        <label
          v-for="[key, value, label] in [
            ['scarf', 'scarf', 'Sciarpa'],
            ['hat', 'cap', 'Cappellino'],
            ['glasses', 'glasses', 'Occhiali'],
            ['flag', 'flag', 'Bandierina'],
            ['shoes', 'shoes', 'Scarpe speciali'],
            ['hair', 'hair', 'Acconciatura alternativa'],
          ]"
          :key="key"
          ><input
            type="checkbox"
            :checked="!!equipment[key]"
            @change="toggle(key, $event.target.checked ? value : '')"
          />{{ label }}</label
        >
      </div>
      <h2>Confronto e animazione</h2>
      <label
        >Livello di dettaglio<select v-model="lod">
          <option value="auto">Automatico · LOD 0 selezionato</option>
          <option value="near">LOD 0 · Ravvicinato</option>
          <option value="far">LOD 1 · Semplificato</option>
        </select></label
      ><label
        ><input v-model="standing" type="checkbox" :disabled="population" />In
        piedi (altrimenti seduto)</label
      >
      <div class="studio-actions">
        <button @click="animate('clap')">Applauso</button
        ><button @click="animate('scarf')">Sciarpa alzata</button
        ><button @click="animate('wave')">Ola</button
        ><button @click="animate('vote')">Esultanza</button>
      </div>
      <label class="stress"
        ><input v-model="population" type="checkbox" />Scena di prova · 340
        avatar</label
      >
      <p>
        Trascina per ruotare a 360°. Rotella o pinch per lo zoom. Le scelte di
        questa pagina restano nel solo laboratorio.
      </p>
      <output aria-label="Statistiche renderer"
        >{{ metrics.fps }} FPS browser · {{ metrics.calls }} draw call<br />{{
          metrics.triangles.toLocaleString("it-IT")
        }}
        triangoli · {{ metrics.detailed }} avatar LOD 0<br />Pose CPU:
        {{ metrics.poseMs }} ms</output
      >
      <p v-if="error" role="alert">{{ error }}</p>
    </aside>
  </main>
</template>
<style scoped>
.avatar-studio input[type="checkbox"] {
  display: inline-block !important;
  appearance: auto !important;
  opacity: 1 !important;
  position: static !important;
  width: 16px !important;
  height: 16px !important;
  min-width: 16px;
  vertical-align: middle;
  margin-right: 6px;
}
.avatar-studio {
  position: fixed;
  inset: 0;
  height: 100dvh;
  overflow: hidden;
  background: #e8e9df;
  color: #2c4133;
  font:
    13px Arial,
    sans-serif;
  z-index: 100;
}
.studio-canvas {
  position: absolute;
  inset: 0 350px 0 0;
  touch-action: none;
}
.studio-canvas :deep(canvas) {
  display: block;
  width: 100%;
  height: 100%;
}
header {
  position: absolute;
  top: 25px;
  left: 28px;
  pointer-events: none;
}
header small {
  font-size: 9px;
  letter-spacing: 1.7px;
  color: #81906e;
}
header h1 {
  font: 600 32px Arial;
  letter-spacing: -1.6px;
  margin: 8px 0;
}
header p {
  font-size: 11px;
  color: #7d8e6c;
}
header a {
  display: inline-block;
  margin-top: 9px;
  color: #607f44;
  font-size: 11px;
  pointer-events: auto;
}
aside {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 350px;
  padding: 23px;
  overflow: auto;
  background: #f5f7ebef;
  border-left: 1px solid #d6dfc9;
  overscroll-behavior: contain;
}
aside h2 {
  font-size: 17px;
  margin: 8px 0 15px;
}
aside > label {
  display: block;
  font-size: 12px;
  margin: 14px 0;
}
aside select {
  display: block;
  width: 100%;
  min-height: 42px;
  background: #fffef7;
  border: 1px solid #cbd6bd;
  border-radius: 9px;
  padding: 8px;
  color: inherit;
  margin-top: 7px;
}
aside input {
  accent-color: #66874a;
}
aside p {
  font-size: 11px;
  line-height: 1.7;
  color: #82916f;
}
.studio-accessories {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin: 15px 0 22px;
  font-size: 11px;
}
.studio-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 7px;
}
.studio-actions button {
  min-height: 40px;
  border: 1px solid #c9d6b4;
  background: #e3edcf;
  color: #426033;
  border-radius: 8px;
  cursor: pointer;
}
output {
  display: block;
  background: #e8efdb;
  padding: 12px;
  font-size: 11px;
  line-height: 1.9;
  border-radius: 10px;
}
.stress {
  padding: 12px;
  background: #e2ebcf;
  border-radius: 9px;
}
button:focus-visible,
a:focus-visible,
select:focus-visible {
  outline: 3px solid #8eae5c;
}
@media (max-width: 700px) {
  .studio-canvas {
    inset: 0 0 48dvh;
  }
  aside {
    top: 52dvh;
    width: 100%;
    padding: 18px;
    border-top: 1px solid #d6dfc9;
    border-left: 0;
    border-radius: 20px 20px 0 0;
  }
  header {
    left: 17px;
    top: 15px;
  }
  header h1 {
    font-size: 24px;
  }
  header small {
    font-size: 7px;
  }
  header p {
    display: none;
  }
  aside select {
    font-size: 16px;
  }
}
</style>
