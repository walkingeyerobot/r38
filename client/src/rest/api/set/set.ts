import { endpoint } from "@/rest/endpoint.ts";
import type { SourceSet } from "@/parse/SourceData.ts";

export const ROUTE_SET = endpoint({
  route: "/api/set/",
  method: "post",
  bodyVars: {
    set: "",
  },
  response: {} as SourceSet,
});
