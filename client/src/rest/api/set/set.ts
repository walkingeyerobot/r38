import { endpoint } from "@/rest/endpoint.ts";
import type { SourceSet } from "@/parse/SourceData.ts";

export const routeSet = endpoint({
  route: "/api/set/",
  method: "post",
  bodyVars: {
    set: "",
  },
  response: {} as SourceSet,
});
