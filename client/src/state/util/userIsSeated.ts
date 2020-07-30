import { DraftState } from '../../draft/DraftState';

export function userIsSeated(id: number | undefined, draft: DraftState) {
  return getPlayerSeat(id, draft) != null;
}

export function getPlayerSeat(id: number | undefined, draft: DraftState) {
  for (let i = 0; i < draft.seats.length; i++) {
    const seat = draft.seats[i];
    if (seat.player?.id == id) {
      return seat;
    }
  }
  return null;
}
