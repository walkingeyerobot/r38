import { DraftState, CardContainer, CardPack } from "./DraftState";
import { checkNotNil } from '../util/checkNotNil';
import { TimelineEvent, TimelineAction, ActionMovePack, PackLocation, PACK_LOCATION_UNUSED, PACK_LOCATION_DEAD } from './TimelineEvent';

export function commitTimelineEvent(event: TimelineEvent, state: DraftState) {
  try {
    for (const action of event.actions) {
      applyAction(action, state);
    }
  } catch (e) {
    console.log('Error while trying to commit event', event);
    throw e;
  }
}

export function rollbackTimelineEvent(event: TimelineEvent, state: DraftState) {
  for (let i = event.actions.length - 1; i >= 0; i--) {
    rollbackAction(event.actions[i], state);
  }
}

function applyAction(action: TimelineAction, state: DraftState) {
  switch (action.type) {
    case 'move-card':
      const srcContainer = checkNotNil(state.packs.get(action.from));
      const dstContainer = checkNotNil(state.packs.get(action.to));
      dstContainer.cards.push(removeCard(action.card, srcContainer));
      break;
    case 'move-pack':
      movePack(action, state, 'forward');
      break;
    case 'announce':
      console.log('ANNOUNCEMENT:', action.message);
      break;
  }
}

function rollbackAction(action: TimelineAction, state: DraftState) {
  switch (action.type) {
    case 'move-card':
      const srcContainer = checkNotNil(state.packs.get(action.to));
      const destContainer = checkNotNil(state.packs.get(action.from));
      destContainer.cards.push(removeCard(action.card, srcContainer));
      break;
    case 'move-pack':
      movePack(action, state, 'reverse');
      break;
    case 'announce':
      break;
  }
}

function removeCard(id: number, container: CardContainer) {
  for (let i = 0; i < container.cards.length; i++) {
    const card = container.cards[i];
    if (card.id == id) {
      container.cards.splice(i, 1);
      return card;
    }
  }
  throw new Error(`Cannot find card ${id} in container ${container}.`);
}

function movePack(
    action: ActionMovePack,
    state: DraftState,
    direction: 'forward' | 'reverse'
) {
  const srcId = direction == 'forward' ? action.from : action.to;
  const dstId = direction == 'forward' ? action.to : action.from;

  const src = getPackLocation(srcId, state);
  const dst = getPackLocation(dstId, state);

  const pack = getPack(action.pack, state);
  if (src != null) {
    removePack(pack.id, src);
  }
  if (dst != null) {
    if (action.insertAction == 'enqueue' && direction == 'forward'
        || action.insertAction == 'unshift' && direction == 'reverse') {
      dst.push(pack);
    } else if (action.insertAction == 'unshift' && direction == 'forward'
        || action.insertAction == 'enqueue' && direction == 'reverse') {
      dst.unshift(pack);
    } else {
      throw new Error(`Unrecognized insertActio n: ${action.insertAction}`);
    }
  }
}

function getPack(id: number, state: DraftState) {
  const pack = state.packs.get(id);
  if (pack == null) {
    console.log('Pack map is', state.packs);
    throw new Error(`Cannot find pack ${id}`);
  }

  if (pack.type != 'pack') {
    throw new Error(`CardContainer ${id} is not a pack`);
  }
  return pack;
}

function getPackLocation(location: PackLocation, state: DraftState) {
  switch (location.seat) {
    case PACK_LOCATION_UNUSED:
      return state.unusedPacks;
    case PACK_LOCATION_DEAD:
      return null;
    default:
      const seat = state.seats[location.seat];
      if (seat == null) {
        throw new Error(`Unknown seat: ${location.seat}.`);
      }
      if (location.queue == 'unopened') {
        return seat.unopenedPacks;
      } else {
        return seat.queuedPacks;
      }
  }
}

function removePack(id: number, packList: CardPack[]) {
  for (let i = 0; i < packList.length; i++) {
    const pack = packList[i];
    if (pack.id == id) {
      packList.splice(i, 1);
      return pack;
    }
  }
  throw new Error(`Cannot find pack ${id} to remove`);
}
