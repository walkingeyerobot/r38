import { SourceData } from './SourceData';
import { parseInitialState } from './parseInitialState';
import { TimelineGenerator } from './TimelineGenerator';
import { DraftState, DraftCard } from '../draft/DraftState';
import { TimelineEvent } from '../draft/TimelineEvent';
import { checkNotNil } from '../util/checkNotNil';

export function parseDraft(
  sourceData: SourceData
): ParsedDraft {
  const state = parseInitialState(sourceData);
  const generator = new TimelineGenerator();
  const { events, isComplete, parseError } =
      generator.generate(state, sourceData.events);

  annotateCards(state, events);

  return {
    name: sourceData.name,
    state,
    events,
    isComplete,
    parseError,
  };
}

export interface ParsedDraft {
  name: string,
  isComplete: boolean,
  state: DraftState,
  events: TimelineEvent[],
  parseError: Error | null,
}

function annotateCards(
  draft: DraftState,
  events: TimelineEvent[],
) {
  const cardMap = new Map<number, DraftCard>();
  for (let pack of draft.packs.values()) {
    for (let card of pack.cards) {
      cardMap.set(card.id, card);
    }
  }

  for (const event of events) {
    for (const action of event.actions) {
      if (action.type == 'move-card' && action.subtype == 'pick-card') {
        const card = checkNotNil(cardMap.get(action.card));
        // TODO: When we clone draft states, this will become a copy rather
        // than a reference
        card.pickedIn.push(event);
      }
    }
  }
}
