import * as THREE from "three";
import { canvasTexture } from "./primitives";
import { arenaLayout } from "./layout";

export function createField(parent) {
  const texture = canvasTexture(2048, 1536, (ctx, w, h) => {
    ctx.fillStyle = "#d8bb77";
    ctx.fillRect(0, 0, w, h);
    // Deterministic maple boards, with the warm, pale finish of the reference.
    let seed = 42;
    const rand = () => {
      seed = (seed * 1664525 + 1013904223) >>> 0;
      return seed / 4294967296;
    };
    for (let row = 0; row < 96; row++) {
      for (let col = -1; col < 15; col++) {
        const x = col * 150 + (row % 3) * 50,
          y = row * 16;
        ctx.fillStyle = `hsl(42, ${34 + rand() * 12}%, ${65 + rand() * 10}%)`;
        ctx.fillRect(x, y, 149, 15);
        ctx.strokeStyle = "rgba(135,102,46,.09)";
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.moveTo(x, y + 5);
        ctx.lineTo(x + 148, y + 5 + rand() * 4);
        ctx.stroke();
      }
    }
    const px = (x) => (x / arenaLayout.length + 0.5) * w,
      pz = (z) => (z / arenaLayout.width + 0.5) * h;
    const line = (points, color = "#fcf7de", width = 4) => {
      ctx.strokeStyle = color;
      ctx.lineWidth = width;
      ctx.beginPath();
      points.forEach(([x, z], i) =>
        i ? ctx.lineTo(px(x), pz(z)) : ctx.moveTo(px(x), pz(z)),
      );
      ctx.stroke();
    };
    line([
      [-9, -4.5],
      [9, -4.5],
      [9, 4.5],
      [-9, 4.5],
      [-9, -4.5],
    ]);
    for (const x of [-3, 0, 3])
      line([
        [x, -4.5],
        [x, 4.5],
      ]);
    line(
      [
        [-14, -7.5],
        [14, -7.5],
        [14, 7.5],
        [-14, 7.5],
        [-14, -7.5],
      ],
      "#e5df99",
      3,
    );
    ctx.strokeStyle = "#ece4ac";
    ctx.lineWidth = 3;
    ctx.beginPath();
    ctx.ellipse(
      px(0),
      pz(0),
      (w * 1.8) / arenaLayout.length,
      (h * 1.8) / arenaLayout.width,
      0,
      0,
      Math.PI * 2,
    );
    ctx.stroke();
    ctx.save();
    ctx.translate(px(0), pz(0));
    ctx.globalAlpha = 0.46;
    ctx.fillStyle = "#5c7693";
    ctx.beginPath();
    ctx.ellipse(0, 0, 73, 57, -0.28, 0, Math.PI * 2);
    ctx.fill();
    ctx.strokeStyle = "#f5edcc";
    ctx.lineWidth = 5;
    ctx.beginPath();
    ctx.arc(0, 0, 39, 0, Math.PI * 2);
    ctx.stroke();
    ctx.font = "bold 22px sans-serif";
    ctx.textAlign = "center";
    ctx.fillStyle = "#fff8dd";
    ctx.fillText("JOY", 0, 7);
    ctx.restore();
  });
  const floor = new THREE.Mesh(
    new THREE.PlaneGeometry(arenaLayout.length, arenaLayout.width),
    new THREE.MeshStandardMaterial({
      map: texture,
      roughness: 0.29,
      metalness: 0.07,
    }),
  );
  floor.rotation.x = -Math.PI / 2;
  floor.position.y = 0.015;
  parent.add(floor);
  // Soft baked pools of overhead light: no costly live reflection pass.
  const glow = canvasTexture(128, 128, (ctx, w, h) => {
    const gradient = ctx.createRadialGradient(
      w / 2,
      h / 2,
      0,
      w / 2,
      h / 2,
      w / 2,
    );
    gradient.addColorStop(0, "rgba(255,255,224,.28)");
    gradient.addColorStop(1, "rgba(255,255,224,0)");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, w, h);
  });
  for (const x of [-9, 0, 9]) {
    const pool = new THREE.Mesh(
      new THREE.PlaneGeometry(7, 14),
      new THREE.MeshBasicMaterial({
        map: glow,
        transparent: true,
        depthWrite: false,
      }),
    );
    pool.rotation.x = -Math.PI / 2;
    pool.position.set(x, 0.025, -2);
    parent.add(pool);
  }
}
