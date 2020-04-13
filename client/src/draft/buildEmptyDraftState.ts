import { DraftState } from './DraftState';

export function buildEmptyDraftState(): DraftState {
  return {
    seats: [],
    unusedPacks: [],
    packs: new Map(),
    isComplete: false,
  }
}
