import { endpoint } from "@/rest/endpoint";
import type { SourceData } from "@/parse/SourceData";

export const ROUTE_MAKE_DRAFT = endpoint({
  route: "/api/makedraft",
  method: "post",
  bodyVars: {
    name: "" as string,
    set: "" as string,
    inPerson: false as boolean,
    assignSeats: false as boolean,
    assignPacks: false as boolean,
    pickTwo: false as boolean,
  },
  queryVars: {
    as: 0,
  } as { as?: number },
  response: {} as SourceData,
});
