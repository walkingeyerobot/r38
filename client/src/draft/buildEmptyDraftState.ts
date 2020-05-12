import { DraftState, PACK_LOCATION_UNUSED, PACK_LOCATION_DEAD } from './DraftState';

export function buildEmptyDraftState(): DraftState {
  return {
    seats: [],
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
