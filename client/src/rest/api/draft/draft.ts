import { endpoint } from "@/rest/endpoint";
import type { SourceData } from "@/parse/SourceData";

export const ROUTE_DRAFT = endpoint({
  route: "/api/draft/:id",
  method: "get",
  pathVars: {} as {
    id: string;
  },
  queryVars: {
    as: 0,
  } as { as?: number },
  response: {} as SourceData,
});
