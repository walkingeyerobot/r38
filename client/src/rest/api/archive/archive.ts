import { endpoint } from "@/rest/endpoint";
import type { SourceData } from "@/parse/SourceData";

export const ROUTE_ARCHIVE_DRAFT = endpoint({
  route: "/api/archive/:id",
  method: "post",
  pathVars: {} as {
    id: string;
  },
  queryVars: {
    as: 0,
  } as { as?: number },
  response: {} as SourceData,
});
