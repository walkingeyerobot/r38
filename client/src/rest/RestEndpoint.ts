/**
 * Defines an HTTP endpoint that the server will serve
 */
export interface RestEndpoint {
  /** A path-to-regexp compatible path, e.g. `/foo/bar/:id` */
  route: string;
  /** The HTTP method to use */
  method: "get" | "post" | "put";
  /* If the route contains any variables, then they should be defined here */
  pathVars: { [key: string]: string };
  /* If the route supports any query vars (e.g. `?foo=blah&bar=47`) */
  queryVars: object | undefined;
  /* If the route is expecting a JSON body, define its members here */
  bodyVars: object | undefined;
  /* The type of the JSON response returned by the server */
  response: object | void;
  /* A function that takes in a pathVars object and returns a bound route */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  routeBinder: (vars: any) => string;
}

/** Same as RestEndpoint, but binds types to the various subfields */
export interface ParsedEndpoint<
  P extends object,
  Q extends object,
  B extends object,
  R extends object | void,
> {
  route: string;
  method: "get" | "post" | "put";
  pathVars: P;
  queryVars: Q;
  bodyVars: B;
  response: R;
  routeBinder: (vars: P) => string;
}
