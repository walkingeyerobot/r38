import { endpoint } from "@/rest/endpoint";

export const ROUTE_USER_INFO = endpoint({
  route: "/api/userinfo/",
  method: "get",
  queryVars: {
    as: 0,
  } as { as?: number },
  response: {} as SourceUserInfo,
});

export interface SourceUserInfo {
  name: string;
  picture: string;
  userId: number;
  mtgoName: string;
}
