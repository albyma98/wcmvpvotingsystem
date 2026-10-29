import { CatmullRomCurve3, MathUtils, Vector3 } from "three";

export const TOUR_SECONDS = 18;

export function createTourSampler() {
  const positions = [
    [-22, 13, 23],
    [-18, 12, 12],
    [-1, 7, 5],
    [3, 9, 7],
    [-12, 6, 1],
    [-19, 10, 17],
  ];
  const targets = [
    [0, 3, -1],
    [3, 3, -2],
    [14, 3, -2],
    [0, 0.5, 0],
    [-4, 1.4, 12],
    [1, 3, -2],
  ];
  const curve = (points) =>
    new CatmullRomCurve3(
      points.map((point) => new Vector3(...point)),
      false,
      "catmullrom",
      0.25,
    );
  const path = curve(positions),
    focus = curve(targets);
  return (elapsedMs) => {
    // A queued RAF may predate the user click by a fraction of a frame.
    const progress = MathUtils.clamp(elapsedMs / (TOUR_SECONDS * 1000), 0, 1);
    const blend = MathUtils.clamp(elapsedMs / 1100, 0, 1);
    return {
      progress,
      ease: blend * blend * (3 - 2 * blend),
      position: path.getPoint(progress),
      target: focus.getPoint(progress),
    };
  };
}
