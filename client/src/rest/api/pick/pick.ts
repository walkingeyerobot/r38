import { endpoint } from "@/rest/endpoint";
import type { SourceData } from "@/parse/SourceData";

export const ROUTE_PICK = endpoint({
  route: "/api/pick/",
  method: "post",
  queryVars: {
    as: 0,
  } as { as?: number },
  bodyVars: {
    draftId: 0,
    cards: [],
    xsrfToken: "",
  } as { draftId: number; cards: number[]; xsrfToken: string },
  response: {} as SourceData,
});
