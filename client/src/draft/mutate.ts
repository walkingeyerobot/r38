import { DraftState, CardContainer, CardPack, PackContainer } from "./DraftState";
import { checkNotNil } from '../util/checkNotNil';
import { TimelineEvent, TimelineAction, ActionMovePack, ActionAssignRound } from './TimelineEvent';
import { MutationError } from './MutationError';

const DEBUG = false;

export function commitTimelineEvent(event: TimelineEvent, state: DraftState) {
  if (DEBUG) {
    console.log(
        'APPLYING EVENT:',
        event.id,
        event.round,
        event.roundEpoch,
        event.associatedSeat,
        event.actions.map(a => a.type));
  }
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
    case 'assign-round':
      assignRound(action, state);
      break;
    default:
      throw new Error(`Unrecognized action ${action}`);
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
    case 'assign-round':
      unassignRound(action, state);
      break;
    default:
      throw new Error(`Unrecognized action ${action}`);
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
  throw new MutationError(`Cannot find card ${id} in container ${container}.`);
}

function movePack(
    action: ActionMovePack,
    state: DraftState,
    direction: 'forward' | 'reverse'
) {
  const src = direction == 'forward' ? action.from : action.to;
  const dst = direction == 'forward' ? action.to : action.from;

  const srcCnt = checkNotNil(state.locations.get(src));
  const dstCnt = checkNotNil(state.locations.get(dst));

  const pack = state.packs.get(action.pack);
  if (pack == undefined || pack.type != 'pack') {
    throw new MutationError(`Cannot find pack ${action.pack}`);
  }

  // TODO: We can't store the specific index that the pack comes from/goes to
  // because this information differs between timeline and synchronized modes.
  // However, this means that we can't properly reverse the event -- we don't
  // know what index position to return the pack to. So for now we require that
  // all pack moves original from index position 0 (i.e. the front of the
  // queue).

  const srcIndex = srcCnt.packs.indexOf(pack);
  const dstIndex = direction == 'forward'
      ? getPackDestinationIndex(pack, dstCnt, action.queuePosition)
      : 0;
  if (direction == 'forward' && srcIndex != 0) {
    throw new MutationError(`Can only move first pack in the container`);
  }

  srcCnt.packs.splice(srcIndex, 1);
  dstCnt.packs.splice(dstIndex, 0, pack);
}

function getPackDestinationIndex(
    pack: CardPack,
    dst: PackContainer,
    queuePosition: 'front' | 'end',
): number {
  let i = 0;
  for (; i < dst.packs.length; i++) {
    if (queuePosition == 'front' && pack.round <= dst.packs[i].round) {
      return i;
    }
    if (queuePosition == 'end' && pack.round < dst.packs[i].round) {
      return i;
    }
  }
  return i;
}

function assignRound(action: ActionAssignRound, state: DraftState) {
  const pack = state.packs.get(action.pack);
  if (pack == undefined || pack.type != 'pack') {
    throw new MutationError(`Not a pack: ${action.pack}`);
  }
  pack.round = action.to;
}

function unassignRound(action: ActionAssignRound, state: DraftState) {
  const pack = state.packs.get(action.pack);
  if (pack == undefined || pack.type != 'pack') {
    throw new MutationError(`Not a pack: ${action.pack}`);
  }
  pack.round = action.from;
}
