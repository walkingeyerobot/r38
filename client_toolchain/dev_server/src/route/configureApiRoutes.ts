import express from 'express';
import { stubEndpoint } from './stubEndpoint';
import { routeDraftlist } from '../../../../client/src/rest/api/draftlist/draftlist';
import { stubDraftlist } from '../../../../client/src/rest/api/draftlist/draftlist.stub';
import { routeDraft } from '../../../../client/src/rest/api/draft/draft';
import { stubDraft_24 } from '../../../../client/src/rest/api/draft/draft_24.stub';
import { stubDraft25_partial } from '../../../../client/src/rest/api/draft/draft_25_partial.stub';
import { routePick } from '../../../../client/src/rest/api/pick/pick';
import { NotFoundError } from '../server/NotFoundError';

export function configureApiRoutes(app: express.Express) {
  stubEndpoint(app, routeDraftlist, stubDraftlist);

  stubEndpoint(app, routeDraft, req => {
    switch (req.params.id) {
      case '24':
        return stubDraft_24;
      case '25':
        return stubDraft25_partial;
      default:
        throw new NotFoundError(`Draft w/ ID ${req.params.id} not found`);
    }
  });

  stubEndpoint(app, routePick, {});
}
