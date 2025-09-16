import { endpoint } from "@/rest/endpoint.ts";
import type { SourcePack } from "@/parse/SourceData.ts";

export const ROUTE_DRAFT_PACKS = endpoint({
  route: "/api/draftpacks/:id",
  method: "get",
  pathVars: {} as {
    id: string;
  },
  queryVars: {
    as: 0,
  } as { as?: number },
  response: {} as SourcePack[],
});
