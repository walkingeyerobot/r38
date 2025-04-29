import { endpoint } from "@/rest/endpoint";

export const ROUTE_JOIN_DRAFT = endpoint({
  method: "post",
  route: "/api/join/",
  queryVars: {
    as: 0,
  } as { as?: number },
  bodyVars: {
    id: 0 as number,
    position: undefined as number | undefined,
  },
  response: {},
});
