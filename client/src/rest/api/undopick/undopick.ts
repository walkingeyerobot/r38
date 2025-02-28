import { endpoint } from "@/rest/endpoint";
import type { SourceData } from "@/parse/SourceData";

export const routeUndoPick = endpoint({
  route: "/api/undopick/",
  method: "post",
  queryVars: {
    as: 0,
  } as { as?: number },
  bodyVars: {
    draftId: 0,
    xsrfToken: "",
  } as { draftId: number; xsrfToken: string },
  response: {} as SourceData,
});
