import { endpoint } from '../../endpoint';
import { SourceData } from '../../../parse/SourceData';

export const routePick = endpoint({
  route: '/api/pick/',
  method: 'post',
  bodyVars: {
    cards: [],
  } as { cards: number[] },
  response: {} as SourceData,
});
