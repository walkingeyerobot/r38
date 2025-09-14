import { endpoint } from "@/rest/endpoint.ts";
import type { SourcePack } from "@/parse/SourceData.ts";

export const ROUTE_SAMPLE_PACK = endpoint({
  route: "/api/samplepack/",
  method: "post",
  bodyVars: {
    set: "",
    seed: 0,
  },
  response: {} as SourcePack,
});
