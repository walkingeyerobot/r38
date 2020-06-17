import { DraftState } from '../DraftState';

export function userIsSeated(id: number | undefined, draft: DraftState) {
  return getUserPosition(id, draft) != -1;
}

export function getUserPosition(id: number | undefined, draft: DraftState) {
  for (let i = 0; i < draft.seats.length; i++) {
    const seat = draft.seats[i];
    if (seat.player?.id == id) {
      return i;
    }
  }
  return -1;
}
