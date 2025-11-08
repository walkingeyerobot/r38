import { endpoint } from "@/rest/endpoint";

export const ROUTE_USER_STATS = endpoint({
  route: "/api/userstats/",
  method: "get",
  queryVars: {
    as: 0,
  } as { as?: number },
  response: {} as SourceUserStats,
});

export interface SourceUserStats {
  activeDrafts: number;
  completedDrafts: number;
}
