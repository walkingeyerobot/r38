import axios, { type AxiosRequestConfig } from "axios";

import type { RestEndpoint } from "@/rest/RestEndpoint";
import type { MixedCollection } from "@/util/MixedCollection";
import type { DefaultEmpty } from "@/util/DefaultEmpty";

export async function fetchEndpoint<T extends RestEndpoint>(
  endpoint: T,
  params: EndpointParams<T>,
): Promise<T["response"]> {
  const response = await axios(buildFetchConfig(endpoint, params));
  return response.data;
}

export type EndpointParams<T extends RestEndpoint> = DefaultEmpty<T["pathVars"]> &
  DefaultEmpty<T["queryVars"]> &
  DefaultEmpty<T["bodyVars"]>;

function buildFetchConfig<T extends RestEndpoint>(
  endpoint: T,
  params: EndpointParams<T>,
): AxiosRequestConfig {
  const config: AxiosRequestConfig = {
    url: endpoint.routeBinder(params),
    method: endpoint.method,
    headers: {
      Accept: "application/json",
    },
  };

  for (const v in endpoint.queryVars) {
    if (config.params == undefined) {
      config.params = {};
    }
    config.params[v] = (params as MixedCollection)[v];
  }

  let hasBodyVars = false;
  const body = {} as MixedCollection;
  for (const v in endpoint.bodyVars) {
    body[v] = (params as MixedCollection)[v];
    hasBodyVars = true;
  }
  if (hasBodyVars) {
    config.data = body;
  }

  return config;
}
