import { endpoint } from "../../endpoint";

export const ROUTE_SKIP_DRAFT = endpoint({
  method: "post",
  route: "/api/skip/",
  response: {},
  bodyVars: {
    id: 0 as number,
  },
});
