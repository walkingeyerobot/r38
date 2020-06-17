import axios, { AxiosRequestConfig } from 'axios';

import { RestEndpoint } from '../rest/RestEndpoint';
import { MixedCollection } from '../util/MixedCollection';
import { DefaultEmpty } from '../util/DefaultEmpty';

export async function fetchEndpoint<T extends RestEndpoint>(
    endpoint: T,
    params: EndpointParams<T>,
): Promise<T['response']> {
  const response = await axios(buildFetchConfig(endpoint, params));
  return response.data;
}

export type EndpointParams<T extends RestEndpoint> =
    & DefaultEmpty<T['pathVars']>
    & DefaultEmpty<T['queryVars']>
    & DefaultEmpty<T['bodyVars']>;


function buildFetchConfig<T extends RestEndpoint>(
    endpoint: T,
    params: EndpointParams<T>,
): AxiosRequestConfig {

  let config: AxiosRequestConfig = {
    url: endpoint.routeBinder(params),
    method: endpoint.method,
    headers: {
      'Accept': 'application/json',
    },
  };

  for (let v in endpoint.queryVars) {
    config.params[v] = (params as MixedCollection)[v];
  }

  let hasBodyVars = false;
  const body = {} as MixedCollection;
  for (let v in endpoint.bodyVars) {
    body[v] = (endpoint.bodyVars as MixedCollection)[v];
  }
  if (hasBodyVars) {
    config.data = body;
  }

  return config;
}
