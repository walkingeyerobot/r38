import { endpoint } from '../../endpoint';

export const routeDraftlist = endpoint({
  route: '/api/draftlist',
  method: 'get',
  response: {} as {
    drafts: HomeDraftDescriptor[];
  },
});

export interface HomeDraftDescriptor {
  id: number;
  name: string;
  availableSeats: number;
  status: 'joinable' | 'member' | 'spectator' | 'closed';
}
