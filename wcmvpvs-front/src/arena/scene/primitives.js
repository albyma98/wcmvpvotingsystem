import * as THREE from "three";

export const palette = {
  concrete: "#d2d0c3",
  riser: "#e5e1d6",
  wall: "#d9d5a3",
  frame: "#eeeae0",
  blue: "#303e83",
  orange: "#e58b32",
  yellow: "#f3ce27",
  wood: "#65442b",
  darkWood: "#3c2b21",
  glass: "#aac7cc",
  chair: "#37434b",
  white: "#f9f6e9",
};

// One instanced draw call per material, rather than one per rail, seat or beam.
export function createBuilder(parent) {
  const batches = new Map();
  const dummy = new THREE.Object3D();
  const up = new THREE.Vector3(0, 1, 0);
  function box(color, position, scale, rotation = [0, 0, 0]) {
    dummy.position.set(...position);
    dummy.scale.set(...scale);
    dummy.rotation.set(...rotation);
    dummy.updateMatrix();
    if (!batches.has(color)) batches.set(color, []);
    batches.get(color).push(dummy.matrix.clone());
  }
  function beam(color, a, b, thickness = 0.06) {
    const start = new THREE.Vector3(...a),
      end = new THREE.Vector3(...b);
    dummy.position.copy(start).add(end).multiplyScalar(0.5);
    dummy.quaternion.setFromUnitVectors(up, end.clone().sub(start).normalize());
    dummy.scale.set(thickness, start.distanceTo(end), thickness);
    dummy.updateMatrix();
    if (!batches.has(color)) batches.set(color, []);
    batches.get(color).push(dummy.matrix.clone());
  }
  function rail(a, b, base = 0, height = 1.05) {
    const length = Math.hypot(b[0] - a[0], b[1] - a[1]);
    const count = Math.ceil(length / 1.8);
    for (let i = 0; i <= count; i++) {
      const t = i / count;
      box(
        palette.blue,
        [a[0] + (b[0] - a[0]) * t, base + height / 2, a[1] + (b[1] - a[1]) * t],
        [0.055, height, 0.055],
      );
    }
    for (const h of [height, height * 0.52])
      beam(palette.blue, [a[0], base + h, a[1]], [b[0], base + h, b[1]], 0.055);
  }
  function flush() {
    const geometry = new THREE.BoxGeometry(1, 1, 1);
    for (const [color, matrices] of batches) {
      const material = new THREE.MeshStandardMaterial({
        color,
        roughness: 0.76,
      });
      const mesh = new THREE.InstancedMesh(geometry, material, matrices.length);
      matrices.forEach((matrix, i) => mesh.setMatrixAt(i, matrix));
      mesh.computeBoundingSphere();
      parent.add(mesh);
    }
    batches.clear();
  }
  return { box, beam, rail, flush };
}

export function canvasTexture(width, height, paint) {
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  paint(canvas.getContext("2d"), width, height);
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.anisotropy = 4;
  return texture;
}

export function sign(
  parent,
  text,
  position,
  width = 3,
  height = 0.65,
  rotation = 0,
) {
  const map = canvasTexture(768, 160, (ctx, w, h) => {
    ctx.fillStyle = "#263931";
    ctx.fillRect(0, 0, w, h);
    ctx.fillStyle = "#f5f3e3";
    ctx.font = "600 66px sans-serif";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(text, w / 2, h / 2);
  });
  const mesh = new THREE.Mesh(
    new THREE.PlaneGeometry(width, height),
    new THREE.MeshBasicMaterial({ map }),
  );
  mesh.position.set(...position);
  mesh.rotation.y = rotation;
  parent.add(mesh);
}
