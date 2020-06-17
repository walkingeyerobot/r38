import express from 'express';
import { RestEndpoint } from "../../../../client/src/rest/RestEndpoint";
import { NotFoundError } from '../server/NotFoundError';

/** Registers an express handler to return [stub] from [endpoint]. */
export function stubEndpoint<T extends RestEndpoint>(
    app: express.Express,
    endpoint: T,
    stub: T['response'] | ((req: express.Request) => T['response']),
) {
  const handler = async (req: express.Request, res: express.Response) => {
    // Add in some fake network delay
    await new Promise(resolve => setTimeout(() => resolve(), 300));

    let json: T['response'];
    if (typeof stub == 'function') {
      try {
        json = stub(req);
      } catch (e) {
        if (e instanceof NotFoundError) {
          res.sendStatus(404);
        } else {
          res.sendStatus(500);
        }
        console.error('Error while serving endpoint', req.path, e.message);
        return;
      }
    } else {
      json = stub;
    }

    res.json(json);
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
