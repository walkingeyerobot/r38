import { type RouteLocationNormalized } from "vue-router";

/**
 * Convenience wrapper for retrieving params from routes. Coerces the raw param
 * type from `string | string[]` to just `string`. Use if you know your param
 * won't be repeated (likely).
 */
export function routeParam(route: RouteLocationNormalized, name: string) {
  const param = route.params[name];
  if (param instanceof Array) {
    return param[0] || "";
  } else {
    return param;
  }
}
