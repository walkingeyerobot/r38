/**
 * Convenience function for creating a tuple-typed array.
 *
 * Normally, using simply an array literal will result in an inferred type of
 * an infinite-length array. For example:
 *
 * var x = ['foo', 3];      // type is Array<string | number>
 * var x = tuple('foo', 3)  // type is [string, number]
 */
export function tuple<T extends any[]>(...elements: T) {
  return elements;
}
