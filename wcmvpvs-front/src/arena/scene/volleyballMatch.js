import * as THREE from "three";
import { createBuilder, palette, canvasTexture } from "./primitives.js";
import { PLAYER_SPOTS, sampleRally, playerPose } from "./volleyballRally.js";

function createNet(parent) {
  const net = new THREE.Group();
  net.name = "volleyball-net";
  parent.add(net);
  const build = createBuilder(net);
  for (const z of [-5.15, 5.15]) {
    build.box("#d9dce0", [0, 1.32, z], [0.09, 2.64, 0.09]);
    build.box(palette.blue, [0, 0.83, z], [0.24, 1.66, 0.24]);
    build.box(
      palette.white,
      [0, 1.54, z - Math.sign(z) * 0.13],
      [0.25, 0.1, 0.025],
    );
    build.beam("#b7bbb5", [0, 2.47, z], [0, 2.47, Math.sign(z) * 4.75], 0.018);
  }
  build.box(palette.white, [0, 2.43, 0], [0.045, 0.075, 9.5]);
  build.box(palette.white, [0, 1.43, 0], [0.035, 0.04, 9.5]);
  for (const z of [-4.75, 4.75])
    build.box(palette.white, [0, 1.93, z], [0.03, 1.02, 0.045]);
  for (const z of [-4.5, 4.5]) {
    for (let stripe = 0; stripe < 8; stripe++)
      build.box(
        stripe % 2 ? "#da5145" : palette.white,
        [0, 2.48 + stripe * 0.1, z],
        [0.025, 0.1, 0.025],
      );
  }
  build.flush();
  const lines = [];
  for (let row = 0; row <= 10; row++)
    lines.push(0, 1.43 + row * 0.1, -4.75, 0, 1.43 + row * 0.1, 4.75);
  for (let col = 0; col <= 95; col++)
    lines.push(0, 1.43, -4.75 + col * 0.1, 0, 2.43, -4.75 + col * 0.1);
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute("position", new THREE.Float32BufferAttribute(lines, 3));
  net.add(
    new THREE.LineSegments(
      geometry,
      new THREE.LineBasicMaterial({
        color: "#343d3b",
        transparent: true,
        opacity: 0.62,
      }),
    ),
  );
}

export function createVolleyballMatch(parent, { reducedMotion = false } = {}) {
  const group = new THREE.Group();
  group.name = "volleyball-match";
  parent.add(group);
  createNet(group);
  // Every body part shares a single instanced box mesh; heads share one sphere.
  const partsPerPlayer = 20;
  const parts = new THREE.InstancedMesh(
    new THREE.BoxGeometry(1, 1, 1),
    new THREE.MeshStandardMaterial({ roughness: 0.86 }),
    12 * partsPerPlayer,
  );
  const heads = new THREE.InstancedMesh(
    new THREE.SphereGeometry(1, 8, 6),
    new THREE.MeshStandardMaterial({ roughness: 0.9 }),
    12,
  );
  parts.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
  heads.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
  parts.frustumCulled = heads.frustumCulled = false;
  group.add(parts, heads);
  const dummy = new THREE.Object3D(),
    a = new THREE.Vector3(),
    b = new THREE.Vector3(),
    up = new THREE.Vector3(0, 1, 0);
  const color = new THREE.Color();
  const skins = [
    "#d6a477",
    "#b57a50",
    "#f0c6a1",
    "#9c6648",
    "#e2b18a",
    "#c68a60",
  ];
  let cursor = 0,
    painting = true;
  let originX = 0,
    originZ = 0,
    baseY = 0,
    forward = 1;
  function point(x, y, z, out) {
    return out.set(originX + x * forward, baseY + y, originZ + z * forward);
  }
  function box(tint, x, y, z, sx, sy, sz) {
    point(x, y, z, dummy.position);
    dummy.rotation.set(0, 0, 0);
    dummy.scale.set(sx, sy, sz);
    dummy.updateMatrix();
    parts.setMatrixAt(cursor, dummy.matrix);
    if (painting) parts.setColorAt(cursor, color.set(tint));
    cursor++;
  }
  function limb(tint, from, to, width) {
    point(...from, a);
    point(...to, b);
    dummy.position.copy(a).add(b).multiplyScalar(0.5);
    dummy.quaternion.setFromUnitVectors(up, b.sub(a).normalize());
    dummy.scale.set(
      width,
      Math.hypot(to[0] - from[0], to[1] - from[1], to[2] - from[2]),
      width,
    );
    dummy.updateMatrix();
    parts.setMatrixAt(cursor, dummy.matrix);
    if (painting) parts.setColorAt(cursor, color.set(tint));
    cursor++;
  }
  const shadowMap = canvasTexture(64, 64, (ctx, w, h) => {
    const gradient = ctx.createRadialGradient(
      w / 2,
      h / 2,
      0,
      w / 2,
      h / 2,
      w / 2,
    );
    gradient.addColorStop(0, "rgba(39,43,29,.27)");
    gradient.addColorStop(1, "rgba(39,43,29,0)");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, w, h);
  });
  const shadows = new THREE.InstancedMesh(
    new THREE.PlaneGeometry(1, 1),
    new THREE.MeshBasicMaterial({
      map: shadowMap,
      transparent: true,
      depthWrite: false,
    }),
    13,
  );
  shadows.frustumCulled = false;
  shadows.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
  group.add(shadows);
  function shadow(index, x, z, width, opacityScale = 1) {
    dummy.position.set(x, 0.035, z);
    dummy.rotation.set(-Math.PI / 2, 0, 0);
    dummy.scale.set(width * opacityScale, width * opacityScale, 1);
    dummy.updateMatrix();
    shadows.setMatrixAt(index, dummy.matrix);
  }
  const ballTexture = canvasTexture(256, 128, (ctx, w, h) => {
    ctx.fillStyle = "#f4d33f";
    ctx.fillRect(0, 0, w, h);
    for (let panel = 0; panel < 6; panel++) {
      ctx.beginPath();
      for (let y = 0; y <= h; y++) {
        const x = (panel * w) / 6 + Math.sin((y / h) * Math.PI * 2) * 12;
        y === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y);
      }
      for (let y = h; y >= 0; y--)
        ctx.lineTo(
          ((panel + 1) * w) / 6 + Math.sin((y / h) * Math.PI * 2) * 12,
          y,
        );
      ctx.closePath();
      ctx.fillStyle = ["#e6bd25", "#274e9e", "#faf6dd"][panel % 3];
      ctx.fill();
      ctx.strokeStyle = "#c3b77b";
      ctx.lineWidth = 1;
      ctx.stroke();
    }
  });
  const ball = new THREE.Mesh(
    new THREE.SphereGeometry(0.18, 16, 12),
    new THREE.MeshStandardMaterial({ map: ballTexture, roughness: 0.55 }),
  );
  ball.name = "rally-ball";
  group.add(ball);
  let time = 0,
    last = null,
    previousFrame = -Infinity;
  function update(now) {
    if (reducedMotion && !painting) return;
    if (last !== null && !reducedMotion)
      time += Math.min(Math.max((now - last) / 1000, 0), 0.08);
    last = now;
    if (now - previousFrame < 1000 / 30) return;
    previousFrame = now;
    const rally = sampleRally(time);
    cursor = 0;
    PLAYER_SPOTS.forEach((spot, index) => {
      const pose = playerPose(index, rally, time, reducedMotion);
      originX = pose.x;
      originZ = pose.z;
      forward = -spot.side;
      baseY = 0.04 + pose.jump - pose.dig * 0.3;
      const shirt = spot.side < 0 ? "#2f4e9d" : "#e8bf28";
      const trim = spot.side < 0 ? "#e9c93c" : "#f7f4e4";
      const skin = skins[index % skins.length];
      box(shirt, 0, 1.23, 0, 0.38, 0.58, 0.52);
      box(trim, 0.201, 1.32, 0, 0.02, 0.11, 0.3);
      box("#f3f0df", 0.213, 1.2, -0.04, 0.02, 0.14, 0.075);
      box("#263b63", 0, 0.88, 0, 0.37, 0.23, 0.46);
      box(skin, 0, 1.59, 0, 0.14, 0.14, 0.17);
      point(0, 1.78, 0, dummy.position);
      dummy.rotation.set(0, 0, 0);
      dummy.scale.set(0.18, 0.22, 0.18);
      dummy.updateMatrix();
      heads.setMatrixAt(index, dummy.matrix);
      if (painting) heads.setColorAt(index, color.set(skin));
      box(
        index % 3 === 0 ? "#322b26" : "#5c3b25",
        -0.035,
        1.94,
        0,
        0.27,
        0.12,
        0.32,
      );
      for (const side of [-1, 1]) {
        const stride = pose.stride * side;
        const knee = [
            0.08 + stride + pose.dig * 0.25,
            0.49 + pose.dig * 0.15,
            side * 0.18,
          ],
          ankle = [stride, 0.12 + pose.dig * 0.3, side * 0.2];
        limb(skin, [0, 0.83, side * 0.15], knee, 0.15);
        limb("#f5f1df", knee, ankle, 0.12);
        box(
          "#f6f3e9",
          stride + 0.07,
          0.07 + pose.dig * 0.3,
          side * 0.2,
          0.3,
          0.13,
          0.18,
        );
        const reach = pose.reach;
        const shoulder = [0, 1.43, side * 0.3];
        const elbow = [
          0.12 + pose.dig * 0.3,
          1.13 + reach * 0.73 - pose.dig * 0.1,
          side * (0.36 - reach * 0.12),
        ];
        const hand = [
          0.21 + pose.dig * 0.47,
          1.02 + reach * 1.02 + pose.dig * 0.25,
          side * (0.3 - reach * 0.13 - pose.dig * 0.23),
        ];
        limb(
          shirt,
          shoulder,
          [elbow[0] * 0.55, 1.34 + reach * 0.2, side * 0.33],
          0.18,
        );
        limb(
          skin,
          [elbow[0] * 0.55, 1.34 + reach * 0.2, side * 0.33],
          elbow,
          0.12,
        );
        limb(skin, elbow, hand, 0.11);
        box(skin, ...hand, 0.14, 0.12, 0.12);
      }
      shadow(index, pose.x, pose.z, 1.12, 1 + pose.jump * 0.28);
    });
    ball.position.set(...rally.ball);
    ball.rotation.set(time * 3.7, 0, time * 2.6);
    shadow(12, rally.ball[0], rally.ball[2], 0.4 + rally.ball[1] * 0.09);
    parts.instanceMatrix.needsUpdate =
      heads.instanceMatrix.needsUpdate =
      shadows.instanceMatrix.needsUpdate =
        true;
    if (painting) {
      parts.instanceColor.needsUpdate = heads.instanceColor.needsUpdate = true;
      painting = false;
    }
  }
  update(0);
  return {
    group,
    update,
    setVisible(value) {
      group.visible = value;
      last = null;
    },
  };
}
