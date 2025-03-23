import { endpoint } from "@/rest/endpoint";

export const routeGetCardPack = endpoint({
  route: "/api/getcardpack/",
  method: "post",
  bodyVars: {
    draftId: 0,
    cardRfid: "",
  } as { draftId: number; cardRfid: string },
  response: {} as { pack: number },
});
