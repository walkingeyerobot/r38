import { endpoint } from "@/rest/endpoint";
import type { SourceData } from "@/parse/SourceData";

export const ROUTE_PICK_RFID = endpoint({
  route: "/api/pickrfid/",
  method: "post",
  queryVars: {
    as: 0,
  } as { as?: number },
  bodyVars: {
    draftId: 0,
    cardRfids: [],
    xsrfToken: "",
  } as { draftId: number; cardRfids: string[]; xsrfToken: string },
  response: {} as SourceData,
});
