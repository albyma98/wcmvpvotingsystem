import * as THREE from "three";
import { createEnvironment, createLighting } from "./environment";
import { createStands } from "./stands";
import { createField } from "./field";
import { createChairs, createProps } from "./props";
import { createCameraSystem } from "../camera/cameraSystem";

export function createArena(container, callbacks) {
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
  const roof = createEnvironment(architecture);
  const railings = createProps(architecture);
  const chairs = createChairs(architecture);
  // Stable extension anchors for GLB seats, sponsor signage and a future crowd.
  for (const name of ["crowd", "sponsors", "interaction-zones"]) {
    const layer = new THREE.Group();
    layer.name = name;
    scene.add(layer);
  }
  const cameras = createCameraSystem(
    camera,
    renderer.domElement,
    callbacks.onTour,
    callbacks.onView,
  );
  let disposed = false,
    hidden = document.hidden,
    last = 0,
    contextLost = false;
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
    renderer.render(scene, camera);
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
    setChairs: (value) => {
      chairs.visible = value;
    },
    setRoof: (value) => {
      roof.visible = value;
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
