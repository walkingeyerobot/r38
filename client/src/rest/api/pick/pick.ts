import { endpoint } from '../../endpoint';

export const routePick = endpoint({
  route: '/api/pick',
  method: 'post',
  bodyVars: {} as {
    cards: number[],
  },
  response: {},
});
