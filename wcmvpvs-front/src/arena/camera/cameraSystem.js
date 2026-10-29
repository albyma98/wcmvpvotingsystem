import * as THREE from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import { createTourSampler } from "./cinematic.js";
export { TOUR_SECONDS } from "./cinematic.js";

export const presets = [
  {
    id: "general",
    label: "Vista generale",
    subtitle: "L’inquadratura della fotografia",
    position: [4, 8.5, 20],
    target: [2, 3.5, -2],
  },
  {
    id: "right",
    label: "Tribuna destra",
    subtitle: "Gradoni e scale arancioni",
    position: [-2, 7, 6],
    target: [14.2, 3.7, -2],
  },
  {
    id: "left",
    label: "Tribuna sinistra",
    subtitle: "Giallo, blu e bordo campo",
    position: [-12, 6, -1],
    target: [-3, 1.5, 12],
  },
  {
    id: "court",
    label: "Campo centrale",
    subtitle: "A un passo dal parquet",
    position: [-11, 6, 8],
    target: [0, 0.5, -1],
  },
  {
    id: "high",
    label: "Angolo alto",
    subtitle: "La geometria del palazzetto",
    position: [-24, 24, 26],
    target: [0, 1, 0],
  },
];

export function createCameraSystem(camera, element, onTour, onView) {
  const controls = new OrbitControls(camera, element);
  controls.enableDamping = true;
  controls.dampingFactor = 0.075;
  controls.minDistance = 5;
  controls.maxDistance = 53;
  controls.minPolarAngle = 0.2;
  controls.maxPolarAngle = Math.PI / 2 - 0.09;
  controls.panSpeed = 0.45;
  controls.rotateSpeed = 0.55;
  controls.zoomSpeed = 0.75;
  controls.screenSpacePanning = false;
  controls.mouseButtons = {
    LEFT: THREE.MOUSE.ROTATE,
    MIDDLE: THREE.MOUSE.DOLLY,
    RIGHT: THREE.MOUSE.PAN,
  };
  let transition = null,
    tour = null,
    active = "general",
    lastProgress = 0;
  const reducedMotion = window.matchMedia(
    "(prefers-reduced-motion: reduce)",
  ).matches;
  const vector = (a) => new THREE.Vector3(...a);
  const sampleTour = createTourSampler();
  function stopTour() {
    if (tour) {
      tour = null;
      controls.enabled = true;
      onTour(false, 0);
    }
  }
  function go(id, instant = false) {
    stopTour();
    const preset = presets.find((view) => view.id === id) || presets[0];
    active = preset.id;
    const end = vector(preset.position);
    // Keep the entire arena in the narrower portrait frame.
    if (id === "general" || id === "high")
      end
        .sub(vector(preset.target))
        .multiplyScalar(Math.max(1, Math.pow(1.05 / camera.aspect, 0.38)))
        .add(vector(preset.target));
    controls.maxDistance = Math.max(
      53,
      end.distanceTo(vector(preset.target)) + 2,
    );
    transition = {
      start: performance.now(),
      from: camera.position.clone(),
      targetFrom: controls.target.clone(),
      to: end,
      targetTo: vector(preset.target),
      duration: instant || reducedMotion ? 0 : 1250,
    };
    onView(preset.id);
  }
  function startTour() {
    transition = null;
    tour = {
      start: performance.now(),
      from: camera.position.clone(),
      targetFrom: controls.target.clone(),
    };
    controls.enabled = false;
    active = "reel";
    onTour(true, 0);
    onView("reel");
  }
  function interaction() {
    transition = null;
    stopTour();
    active = "custom";
    onView("custom");
  }
  controls.addEventListener("start", interaction);
  function update(now) {
    if (tour) {
      const { progress, ease, position, target } = sampleTour(now - tour.start);
      camera.position.lerpVectors(tour.from, position, ease);
      controls.target.lerpVectors(tour.targetFrom, target, ease);
      camera.lookAt(controls.target);
      if (now - lastProgress > 80 || progress === 1) {
        onTour(true, progress);
        lastProgress = now;
      }
      if (progress === 1) {
        stopTour();
        active = "custom";
        onView("custom");
      }
    } else if (transition) {
      const t =
        transition.duration === 0
          ? 1
          : THREE.MathUtils.clamp(
              (now - transition.start) / transition.duration,
              0,
              1,
            );
      const ease = t * t * (3 - 2 * t);
      camera.position.lerpVectors(transition.from, transition.to, ease);
      controls.target.lerpVectors(
        transition.targetFrom,
        transition.targetTo,
        ease,
      );
      if (t === 1) transition = null;
    }
    if (!tour) {
      controls.target.x = THREE.MathUtils.clamp(controls.target.x, -14, 16);
      controls.target.z = THREE.MathUtils.clamp(controls.target.z, -12, 13);
      controls.target.y = THREE.MathUtils.clamp(controls.target.y, 0.5, 6);
      controls.update();
      if (camera.position.y < 1.3) camera.position.y = 1.3;
    }
  }
  return {
    go,
    startTour,
    stopTour,
    update,
    reframe: () => {
      if (active === "general" || active === "high") go(active, true);
    },
    dispose: () => {
      controls.removeEventListener("start", interaction);
      controls.dispose();
    },
  };
}
