import express from 'express';
import { createProxyMiddleware } from 'http-proxy-middleware';

import { stubEndpoint } from './stubEndpoint';
import { routeDraftlist } from '../../../../client/src/rest/api/draftlist/draftlist';
import { stubDraftlist } from '../../../../client/src/rest/api/draftlist/draftlist.stub';
import { routeDraft } from '../../../../client/src/rest/api/draft/draft';
import { routePick } from '../../../../client/src/rest/api/pick/pick';
import { NotFoundError } from '../server/NotFoundError';

export function configureApiRoutes(app: express.Express, proxy: boolean) {
  if (proxy) {
    app.use('/api', createProxyMiddleware({
      target: 'http://localhost:12270',
      changeOrigin: true,
    }));
  } else {
    configureStubbedRoutes(app);
  }
}

function configureStubbedRoutes(app: express.Express) {
  stubEndpoint(app, routeDraftlist, stubDraftlist);

  stubEndpoint(app, routeDraft, req => {
    throw new NotFoundError(`Draft w/ ID ${req.params.id} not found`);
  });

  stubEndpoint(app, routePick, req => {
    throw new NotFoundError(`Draft w/ ID ${req.params.id} not found`);
  });
}
