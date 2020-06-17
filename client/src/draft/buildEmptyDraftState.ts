import { DraftState, PACK_LOCATION_UNUSED, PACK_LOCATION_DEAD, CONTAINER_SHADOW } from './DraftState';

export function buildEmptyDraftState(): DraftState {
  return {
    seats: [],
    shadowPool: {
      type: 'shadow-realm',
      id: CONTAINER_SHADOW,
      owningSeat: -1,
      cards: [],
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
