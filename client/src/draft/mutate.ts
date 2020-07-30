import { DraftState, CardContainer, CardPack } from "./DraftState";
import { checkNotNil } from '../util/checkNotNil';
import { TimelineEvent, TimelineAction, ActionMovePack, ActionAssignPackRound, ActionIncrementSeatRound } from './TimelineEvent';
import { MutationError } from './MutationError';
import { CardStore } from './CardStore';
import { eventToString } from '../state/util/eventToString';
import { checkExhaustive } from '../util/checkExhaustive';

const DEBUG = false;

export function commitTimelineEvent(
    cardStore: CardStore,
    event: TimelineEvent,
    state: DraftState,
) {
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
      applyAction(cardStore, action, state);
    }
  } catch (e) {
    console.log('Error while trying to commit event:');
    console.log(eventToString(event));
    throw e;
  }
}

export function rollbackTimelineEvent(
    cardStore: CardStore,
    event: TimelineEvent,
    state: DraftState,
) {
  try {
    for (let i = event.actions.length - 1; i >= 0; i--) {
      rollbackAction(cardStore, event.actions[i], state);
    }
  } catch (e) {
    console.log('Error while trying to rollback event:');
    console.log(eventToString(event));
    throw e;
  }
}

function applyAction(
    cardStore: CardStore,
    action: TimelineAction,
    state: DraftState,
) {
  switch (action.type) {
    case 'move-card':
      moveCard(state, cardStore, action.card, action.from, action.to);
      break;
    case 'move-pack':
      movePack(action, state, 'forward');
      break;
    case 'announce':
      console.log('ANNOUNCEMENT:', action.message);
      break;
    case 'increment-seat-round':
      state.seats[action.seat].round++;
      break;
    case 'assign-pack-round':
      assignPackRound(action, state);
      break;
    case 'mark-transfer':
      markTransfer(state, action.from, action.to);
      break;
    default:
      checkExhaustive(action);
  }
}

function rollbackAction(
    cardStore: CardStore,
    action: TimelineAction,
    state: DraftState,
) {
  switch (action.type) {
    case 'move-card':
      moveCard(state, cardStore, action.card, action.to, action.from);
      break;
    case 'move-pack':
      movePack(action, state, 'reverse');
      break;
    case 'announce':
      break;
    case 'increment-seat-round':
      state.seats[action.seat].round--;
      break;
    case 'assign-pack-round':
      unassignPackRound(action, state);
      break;
    case 'mark-transfer':
      markTransfer(state, action.to, action.from);
      break;
    default:
      checkExhaustive(action);
  }
}

function removeCard(
    cardStore: CardStore,
    id: number,
    container: CardContainer,
) {
  for (let i = 0; i < container.cards.length; i++) {
    const cardId = container.cards[i];
    if (cardId == id) {
      container.cards.splice(i, 1);
      return cardId;
    }
  }
  throw new MutationError(
      `Cannot find card ${id} ${cardStore.getCard(id).definition.name} in `
          + `container ${container.id} w/ contents `
          +  container.cards.map(
                id => id + ':' + cardStore.getCard(id).definition.name));
}

function moveCard(
    state: DraftState,
    cardStore: CardStore,
    card: number,
    from: number,
    to: number,
) {
  const srcContainer = checkNotNil(state.packs.get(from));
  const dstContainer = checkNotNil(state.packs.get(to));
  dstContainer.cards.push(removeCard(cardStore, card, srcContainer));
}

function movePack(
    action: ActionMovePack,
    state: DraftState,
    direction: 'forward' | 'reverse'
) {
  const pack = state.packs.get(action.pack);
  if (pack == undefined || pack.type != 'pack') {
    throw new MutationError(`Cannot find pack ${action.pack}`);
  }

  const src = direction == 'forward' ? action.from : action.to;
  const dst = direction == 'forward' ? action.to : action.from;
  const srcCnt = checkNotNil(state.locations.get(src));
  const dstCnt = checkNotNil(state.locations.get(dst));


  if (action.epoch == 'increment') {
    if (direction == 'forward') {
      pack.epoch++;
    } else {
      pack.epoch--;
    }
  } else {
    if (direction == 'forward') {
      pack.epoch = action.epoch;
    } else {
      pack.epoch = 0;
    }
  }

  const srcIndex = srcCnt.packs.indexOf(pack);
  srcCnt.packs.splice(srcIndex, 1);
  dstCnt.packs.push(pack);
  dstCnt.packs.sort(packComparator);
}

function markTransfer(state: DraftState, from: number, to: number) {
  const srcContainer = checkNotNil(state.packs.get(from));
  const dstContainer = checkNotNil(state.packs.get(to));
  srcContainer.count--;
  dstContainer.count++;
}

function assignPackRound(action: ActionAssignPackRound, state: DraftState) {
  const pack = state.packs.get(action.pack);
  if (pack == undefined || pack.type != 'pack') {
    throw new MutationError(`Not a pack: ${action.pack}`);
  }
  pack.round = action.to;
}

function unassignPackRound(action: ActionAssignPackRound, state: DraftState) {
  const pack = state.packs.get(action.pack);
  if (pack == undefined || pack.type != 'pack') {
    throw new MutationError(`Not a pack: ${action.pack}`);
  }
  pack.round = action.from;
}

function packComparator(a: CardPack, b: CardPack) {
  if (a.round != b.round) {
    return a.round - b.round;
  } else {
    return a.epoch - b.epoch;
  }
}
