import { CatmullRomCurve3, MathUtils, Vector3 } from "three";

export const TOUR_SECONDS = 18;

export function createTourSampler() {
  const positions = [
    [25, 17, 27],
    [23, 13, 13],
    [-3, 8, 4],
    [0, 9, 1],
    [3, 8, -5],
    [22, 12, 23],
  ];
  const targets = [
    [0, 3, 0],
    [0, 3, -3],
    [0, 3, -12],
    [0, 0.5, 0],
    [0, 3, 12],
    [0, 3, 0],
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
