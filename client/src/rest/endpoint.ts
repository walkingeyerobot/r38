import { compile } from "path-to-regexp";
import type { ParsedEndpoint } from "./RestEndpoint";
import type { DefaultEmpty } from "@/util/DefaultEmpty";

export function endpoint<T extends EndpointDescriptor>(
  descriptor: T,
): ParsedEndpoint<
  DefaultEmpty<T["pathVars"]>,
  DefaultEmpty<T["queryVars"]>,
  DefaultEmpty<T["bodyVars"]>,
  T["response"]
> {
  return {
    route: descriptor.route,
    method: descriptor.method,
    pathVars: (descriptor.pathVars || {}) as DefaultEmpty<T["pathVars"]>,
    queryVars: (descriptor.queryVars || {}) as DefaultEmpty<T["queryVars"]>,
    bodyVars: (descriptor.bodyVars || {}) as DefaultEmpty<T["bodyVars"]>,
    response: descriptor.response,
    routeBinder: compile(descriptor.route),
  };
}

export function dynamicEndpoint<T extends EndpointDescriptor>(
  descriptor: T,
  routeBinder: (params: T["pathVars"]) => string,
): ParsedEndpoint<
  DefaultEmpty<T["pathVars"]>,
  DefaultEmpty<T["queryVars"]>,
  DefaultEmpty<T["bodyVars"]>,
  T["response"]
> {
  return {
    route: descriptor.route,
    method: descriptor.method,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    pathVars: descriptor.pathVars || ({} as any),
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    queryVars: descriptor.queryVars || ({} as any),
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    bodyVars: descriptor.bodyVars || ({} as any),
    response: descriptor.response,
    routeBinder,
  };
}

export interface EndpointDescriptor {
  route: string;
  method: "get" | "post" | "put";
  pathVars?: object;
  queryVars?: object;
  bodyVars?: object;
  response: object | void;
}
