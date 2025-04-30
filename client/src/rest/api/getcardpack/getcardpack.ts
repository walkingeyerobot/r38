import { endpoint } from "@/rest/endpoint";

export const ROUTE_GET_CARD_PACK = endpoint({
  route: "/api/getcardpack/",
  method: "post",
  bodyVars: {
    draftId: 0,
    cardRfid: "",
  } as { draftId: number; cardRfid: string },
  response: {} as { pack: number },
});
