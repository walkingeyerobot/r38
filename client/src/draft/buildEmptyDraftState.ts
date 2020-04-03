import { DraftState } from './draft_types';

export function buildEmptyDraftState(): DraftState {
  return {
    seats: [],
    unusedPacks: [],
    packs: new Map(),
  }
}
