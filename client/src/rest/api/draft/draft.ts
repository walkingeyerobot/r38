import { endpoint } from '../../endpoint';
import { SourceData } from '../../../parse/SourceData';

export const routeDraft = endpoint({
  route: '/api/draft/:id',
  method: 'get',
  pathVars: {} as {
    id: number;
  },
  queryVars: {
    as: 0,
  } as { as?: number },
  response: {} as SourceData,
});
