import { endpoint } from "@/rest/endpoint";

export const ROUTE_JOIN_DRAFT = endpoint({
  method: "post",
  route: "/api/join/",
  response: {},
  bodyVars: {
    id: 0 as number,
    position: undefined as number | undefined,
  },
});
