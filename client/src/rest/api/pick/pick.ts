import { endpoint } from "@/rest/endpoint";
import type { SourceData } from "@/parse/SourceData";

export const routePick = endpoint({
  route: "/api/pick/",
  method: "post",
  queryVars: {
    as: 0,
  } as { as?: number },
  bodyVars: {
    cards: [],
    xsrfToken: "",
  } as { cards: number[]; xsrfToken: string },
  response: {} as SourceData,
});
