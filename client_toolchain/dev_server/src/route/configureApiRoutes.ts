import express from 'express';
import { stubEndpoint } from './stubEndpoint';
import { routeDraftlist } from '../../../../client/src/rest/api/draftlist/draftlist';
import { stubDraftlist } from '../../../../client/src/rest/api/draftlist/draftlist.stub';
import { routeDraft } from '../../../../client/src/rest/api/draft/draft';
import { stubDraft_17 } from '../../../../client/src/rest/api/draft/draft.17.stub';

export function configureApiRoutes(app: express.Express) {
  stubEndpoint(app, routeDraftlist, stubDraftlist)
  stubEndpoint(app, routeDraft, stubDraft_17);
}
