import { DraftState, CardPack } from '../../draft/DraftState';
import { checkNotNil } from '../../util/checkNotNil';

export function getPack(draft: DraftState, id: number): CardPack {
  const pack = checkNotNil(draft.packs.get(id));
  if (pack.type != 'pack') {
    throw new Error(`Pack ${id} is not a pack, it's a ${pack.type}`);
  }
  return pack;
}

/**
 * Returns the pack that a player is currently looking at, if any.
 */
export function getActivePackForSeat(draft: DraftState, seatPosition: number) {
  const seat = checkNotNil(draft.seats[seatPosition]);
  const pack = seat.queuedPacks.packs[0];

  if (pack == null || pack.round != seat.round) {
    return null;
  } else {
    return pack;
  }
}

export function getSeat(draft: DraftState, id: number) {
  return checkNotNil(draft.seats[id]);
}
