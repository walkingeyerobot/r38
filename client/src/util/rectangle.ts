export interface Point {
  x: number,
  y: number,
}

export interface Rectangle {
  start: Point,
  end: Point,
}

export function intersects(a: Rectangle, b: Rectangle): boolean {
  const aLeft = Math.min(a.start.x, a.end.x);
  const aRight = Math.max(a.start.x, a.end.x);
  const aTop = Math.min(a.start.y, a.end.y);
  const aBottom = Math.max(a.start.y, a.end.y);
  const bLeft = Math.min(b.start.x, b.end.x);
  const bRight = Math.max(b.start.x, b.end.x);
  const bTop = Math.min(b.start.y, b.end.y);
  const bBottom = Math.max(b.start.y, b.end.y);
  if (aRight < bLeft) {
    return false;
  }
  if (aLeft > bRight) {
    return false;
  }
  if (aBottom < bTop) {
    return false;
  }
  if (aTop > bBottom) {
    return false;
  }
  return true;
}
