import { stub } from '../../stub';
import { routeDraftlist } from './draftlist';

export const stubDraftlist = stub(routeDraftlist, {
  drafts: [
    {
      id: 11,
      name: 'mtgo draft 1',
      status: 'spectator',
      availableSeats: 0,
    },
    {
      id: 12,
      name: 'mtgo draft 2',
      status: 'member',
      availableSeats: 0,
    },
    {
      id: 13,
      name: 'mtgo draft 3',
      status: 'member',
      availableSeats: 0,
    },
    {
      id: 14,
      name: 'closed draft',
      status: 'closed',
      availableSeats: 3,
    },
    {
      id: 15,
      name: 'joinable draft',
      status: 'joinable',
      availableSeats: 6,
    },
  ],
});
