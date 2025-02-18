import { endpoint } from "@/rest/endpoint";

export const routePrefs = endpoint({
  route: "/api/prefs/",
  method: "get",
  queryVars: {},
  response: {} as {
    prefs: UserPrefDescriptor[];
  },
});

export const routeSetPref = endpoint({
  route: "/api/setpref/",
  method: "post",
  bodyVars: {
    pref: {} as UserPrefDescriptor | undefined,
    mtgoName: "" as string | undefined,
  },
  response: {},
});

export interface UserPrefDescriptor {
  format: string;
  elig: boolean;
  name: string;
}
