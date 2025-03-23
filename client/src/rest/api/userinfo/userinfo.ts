import { endpoint } from "@/rest/endpoint";

export const routeUserInfo = endpoint({
  route: "/api/userinfo/",
  method: "get",
  response: {} as SourceUserInfo,
});

export interface SourceUserInfo {
  name: string;
  picture: string;
  userId: number;
  mtgoName: string;
}
