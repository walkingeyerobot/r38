import { DraftState, PACK_LOCATION_UNUSED, PACK_LOCATION_DEAD } from './DraftState';

export function buildEmptyDraftState(): DraftState {
  return {
    seats: [],
    shadowSeat: {
      position: -1,
      player: {
        id: -1,
        name: 'The Shadow',
        iconUrl: 'none',
        seatPosition: -1,
        picks: {
          type: 'seat',
          id: -1,
          cards: [],
        },
      },
      queuedPacks: { id: 0, packs: [], label: '' },
      unopenedPacks: { id: 1, packs: [], label: '' },
      originalPacks: [],
    },
    unusedPacks: {
      id: PACK_LOCATION_UNUSED,
      packs: [],
      label: 'Unused packs',
    },
    deadPacks: {
      id: PACK_LOCATION_DEAD,
      packs: [],
      label: 'Dead packs',
    },
    packs: new Map(),
    locations: new Map(),
  };
}
