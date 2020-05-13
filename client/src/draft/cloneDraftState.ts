import { DraftState } from './DraftState';
import { fillDraftStateMaps } from '../parse/fillDraftStateMaps';

export function cloneDraftState(src: DraftState) {
  const clonedState: DraftState = JSON.parse(JSON.stringify(src));

  fillDraftStateMaps(clonedState);

  return clonedState;
}
