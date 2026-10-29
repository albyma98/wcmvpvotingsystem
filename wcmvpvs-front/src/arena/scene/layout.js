// X follows the 18 m length of the volleyball court; Z follows its 9 m width.
// Both stands run parallel to X. The end walls at ±X remain clear of stands.
export const arenaLayout = {
  length: 36,
  width: 35,
  endWallX: 18.3,
  sideWallZ: 17.5,
  stand: {
    length: 23,
    rows: 19,
    rowDepth: 0.38,
    rowRise: 0.31,
    frontZ: 9.1,
    aisles: [-7.2, 6.7],
  },
  doorOffsets: [-5.5, 5.5],
};
