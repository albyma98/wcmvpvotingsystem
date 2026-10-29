import * as THREE from "three";
import { createEnvironment, createLighting } from "./environment";
import { createStands } from "./stands";
import { createField } from "./field";
import { createChairs, createProps } from "./props";
import { createCameraSystem } from "../camera/cameraSystem";
import { createCrowd } from "../crowd/createCrowd.js";

export function createArena(container, callbacks, initialCrowd) {
  const scene = new THREE.Scene();
  scene.background = new THREE.Color("#e9e8e0");
  const renderer = new THREE.WebGLRenderer({
    antialias: true,
    alpha: false,
    powerPreference: "high-performance",
  });
  renderer.setPixelRatio(
    Math.min(
      window.devicePixelRatio || 1,
      window.innerWidth < 700 ? 1.25 : 1.75,
    ),
  );
  renderer.outputColorSpace = THREE.SRGBColorSpace;
  renderer.toneMapping = THREE.ACESFilmicToneMapping;
  renderer.toneMappingExposure = 1.15;
  renderer.domElement.setAttribute(
    "aria-label",
    "Palazzetto 3D interattivo: trascina per ruotare, usa la rotella o due dita per lo zoom. I pulsanti consentono di scegliere le viste anche da tastiera.",
  );
  renderer.domElement.setAttribute("role", "img");
  container.appendChild(renderer.domElement);
  const camera = new THREE.PerspectiveCamera(57, 1, 0.15, 180);
  camera.position.set(-22, 13, 23);
  createLighting(scene);
  const architecture = new THREE.Group();
  architecture.name = "arena-geometry";
  scene.add(architecture);
  createField(architecture);
  createStands(architecture);
  const { roof, walls } = createEnvironment(architecture);
  const railings = createProps(architecture);
  const chairs = createChairs(architecture);
  // Stable extension anchors for GLB seats, sponsor signage and a future crowd.
  for (const name of ["sponsors", "interaction-zones"]) {
    const layer = new THREE.Group();
    layer.name = name;
    scene.add(layer);
  }
  const crowd = initialCrowd
    ? createCrowd(scene, initialCrowd.actors, {
        selectedId: initialCrowd.selectedId,
        onSelect: callbacks.onSelectProfile,
      })
    : null;
  const flags = scene.getObjectByName("flags-above-bare-stand");
  const reducedMotion = window.matchMedia(
    "(prefers-reduced-motion: reduce)",
  ).matches;
  const cameras = createCameraSystem(
    camera,
    renderer.domElement,
    callbacks.onTour,
    callbacks.onView,
  );
  let disposed = false,
    hidden = document.hidden,
    last = 0,
    contextLost = false,
    roofEnabled = true;
  function resize() {
    const { width, height } = container.getBoundingClientRect();
    if (!width || !height) return;
    camera.aspect = width / height;
    camera.updateProjectionMatrix();
    renderer.setSize(width, height, false);
    cameras.reframe();
  }
  const observer = new ResizeObserver(resize);
  observer.observe(container);
  resize();
  cameras.go("general", true);
  const visibility = () => {
    hidden = document.hidden;
    if (hidden) cameras.stopTour();
  };
  let pointerStart = null;
  const pointers = new Set();
  const pointerDown = (event) => {
    pointers.add(event.pointerId);
    if (pointers.size !== 1 || event.button > 0) {
      pointerStart = null;
      return;
    }
    pointerStart = {
      x: event.clientX,
      y: event.clientY,
      time: performance.now(),
    };
  };
  const pointerUp = (event) => {
    if (
      pointers.size === 1 &&
      pointerStart &&
      performance.now() - pointerStart.time < 500 &&
      Math.hypot(
        event.clientX - pointerStart.x,
        event.clientY - pointerStart.y,
      ) < 6
    )
      crowd?.pick(event.clientX, event.clientY, camera, renderer.domElement);
    pointerStart = null;
    pointers.delete(event.pointerId);
  };
  const pointerCancel = (event) => {
    pointers.delete(event.pointerId);
    pointerStart = null;
  };
  renderer.domElement.addEventListener("pointerdown", pointerDown);
  renderer.domElement.addEventListener("pointerup", pointerUp);
  renderer.domElement.addEventListener("pointercancel", pointerCancel);
  const marker = new THREE.Vector3();
  let lastMarker = 0;
  document.addEventListener("visibilitychange", visibility);
  const lost = (event) => {
    event.preventDefault();
    contextLost = true;
    callbacks.onError(
      "La sessione 3D è stata interrotta dal dispositivo. Ricarica la demo per ripartire.",
    );
  };
  renderer.domElement.addEventListener("webglcontextlost", lost);
  renderer.setAnimationLoop((now) => {
    if (disposed || hidden || contextLost) return;
    cameras.update(now);
    crowd?.update(now, camera);
    if (flags && !reducedMotion)
      flags.rotation.x = Math.sin(now / 1600) * 0.008;
    // Exterior views open the entire near wall, including its frames and doors.
    for (const { group, axis, side, limit } of walls) {
      group.visible = side * camera.position[axis] <= limit + 0.05;
    }
    const roofHeight = 16.7 - (Math.abs(camera.position.z) * 4.3) / 18;
    roof.visible = roofEnabled && camera.position.y < roofHeight;
    renderer.render(scene, camera);
    if (crowd && now - lastMarker > 100) {
      const position = crowd.selectedPosition();
      if (position) {
        marker.copy(position).project(camera);
        callbacks.onMarker?.({
          x: (marker.x + 1) * 50,
          y: (1 - marker.y) * 50,
          visible:
            marker.z > -1 &&
            marker.z < 1 &&
            Math.abs(marker.x) < 0.76 &&
            Math.abs(marker.y) < 0.8,
        });
      }
      lastMarker = now;
    }
    if (now - last > 1000) {
      callbacks.onStats({
        calls: renderer.info.render.calls,
        chairs: chairs.userData.count,
      });
      last = now;
    }
  });
  return {
    view: (id) => cameras.go(id),
    startTour: cameras.startTour,
    stopTour: cameras.stopTour,
    focusFan(id) {
      const actor = crowd?.actor(id);
      if (actor) {
        crowd.select(id);
        cameras.focus(actor.seat);
      }
    },
    selectFan: (id) => crowd?.select(id),
    fanAction(kind, id) {
      if (kind === "wave") cameras.go("general");
      else {
        const actor = crowd?.actor(id);
        if (actor) cameras.focus(actor.seat);
      }
      return crowd?.trigger(kind, id);
    },
    setCrowdFilters: (options) => crowd?.setFilters(options),
    resetCrowd: () => crowd?.reset(),
    setChairs: (value) => {
      chairs.visible = value;
    },
    setRoof: (value) => {
      roofEnabled = value;
    },
    setRailings: (value) => {
      railings.visible = value;
    },
    dispose() {
      disposed = true;
      renderer.setAnimationLoop(null);
      observer.disconnect();
      cameras.dispose();
      document.removeEventListener("visibilitychange", visibility);
      renderer.domElement.removeEventListener("webglcontextlost", lost);
      renderer.domElement.removeEventListener("pointerdown", pointerDown);
      renderer.domElement.removeEventListener("pointerup", pointerUp);
      renderer.domElement.removeEventListener("pointercancel", pointerCancel);
      const geometries = new Set(),
        materials = new Set(),
        textures = new Set();
      scene.traverse((object) => {
        if (object.geometry) geometries.add(object.geometry);
        const list = object.material
          ? Array.isArray(object.material)
            ? object.material
            : [object.material]
          : [];
        list.forEach((material) => {
          materials.add(material);
          Object.values(material).forEach((value) => {
            if (value?.isTexture) textures.add(value);
          });
        });
        if (object.isInstancedMesh) object.dispose();
      });
      textures.forEach((texture) => texture.dispose());
      materials.forEach((material) => material.dispose());
      geometries.forEach((geometry) => geometry.dispose());
      renderer.dispose();
      renderer.domElement.remove();
    },
  };
}
