import { endpoint } from "@/rest/endpoint";
import type { SourceData } from "@/parse/SourceData";

export const ROUTE_TOGGLE_IN_PERSON = endpoint({
  route: "/api/dev/toggleInPerson/:id",
  method: "post",
  pathVars: {} as {
    id: string;
  },
  queryVars: {
    as: 0,
  } as { as?: number },
  response: {} as SourceData,
});
