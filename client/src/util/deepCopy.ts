/**
 * Performs a deep copy of a simple object and its descendants
 *
 * - Circular references are preserved
 * - Multi-references are preserved (e.g. the same object referenced multiple
 *   times within the hierarchy).
 * - Standard collections (Array, Map, and Set) are properly duplicated.
 * - Some built-in complex types are supported (Regexp and Date).
 *
 * Like any deep copy, this one comes with significant caveats:
 * - Does not copy complex objects, i.e. objects whose immediate prototype is
 *   not Object.prototype. These are replaced with {}.
 * - Only enumerable properties are copied.
 * - Property descriptors are all discarded (getters, setters, configurable,
 *   etc.).
 * - Functions are reused from the original. If those functions capture scope
 *   in their closures, so be it.
 */
export function deepCopy<T>(src: T): T {
  return dc(src, new Map<object, object>());
}

function dc<T>(src: T, seen: Map<object, object>): T {
  if (!isObject(src)) {
    return src;
  }

  if (seen.has(src)) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return seen.get(src) as any;
  }

  let dst: object;

  if (Array.isArray(src)) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const arr: any[] = [];
    seen.set(src, arr);
    for (const child of src) {
      arr.push(dc(child, seen));
    }
    dst = arr;
  } else if (src instanceof Map) {
    const map = new Map();
    seen.set(src, map);
    for (const key of src.keys()) {
      map.set(dc(key, seen), dc(src.get(key), seen));
    }
    dst = map;
  } else if (src instanceof Set) {
    const set = new Set();
    seen.set(src, set);
    for (const member of set.values()) {
      set.add(dc(member, seen));
    }
    dst = set;
  } else if (src instanceof RegExp) {
    dst = new RegExp(src.toString());
    seen.set(src, dst);
  } else if (src instanceof Date) {
    dst = new Date(src.getTime());
    seen.set(src, dst);
  } else if (Object.getPrototypeOf(src) != Object.prototype) {
    // Unsupported class instance; replace with {}
    dst = {};
    seen.set(src, dst);
  } else {
    const obj = {} as T & object;
    seen.set(src, obj);
    for (const v in src) {
      obj[v] = dc(src[v], seen);
    }
    dst = obj;
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return dst as any;
}

function isObject(x: unknown): x is object {
  return typeof x == "object" && x != null;
}
