import { DraftState, CardPack } from '../../draft/DraftState';
import { checkNotNil } from '../../util/checkNotNil';

export function getPack(draft: DraftState, id: number): CardPack {
  const pack = checkNotNil(draft.packs.get(id));
  if (pack.type != 'pack') {
    throw new Error(`Pack ${id} is not a pack, it's a ${pack.type}`);
  }
  return pack;
}

export function getSeat(draft: DraftState, id: number) {
  return checkNotNil(draft.seats[id]);
}
