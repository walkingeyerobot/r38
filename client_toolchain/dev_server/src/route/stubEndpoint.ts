import express from 'express';
import { RestEndpoint } from "../../../../client/src/rest/RestEndpoint";

/** Registers an express handler to return [stub] from [endpoint]. */
export function stubEndpoint<T extends RestEndpoint>(
    app: express.Express,
    endpoint: T,
    stub: T['response'],
) {
  const handler = (req: express.Request, res: express.Response) => {
    res.json(stub);
  }

  switch (endpoint.method) {
    case 'get':
      app.get(endpoint.route, handler);
      break;
    case 'post':
      app.post(endpoint.route, handler);
      break;
    case 'put':
      app.put(endpoint.route, handler);
      break;
    default:
      throw new Error(`Unrecognized method: ${endpoint.method}`);
  }
}
