import { SourceData } from './SourceData';
import { parseInitialState } from './parseInitialState';
import { TimelineGenerator } from './TimelineGenerator';
import { DraftState } from '../draft/DraftState';
import { TimelineEvent } from '../draft/TimelineEvent';

export function parseDraft(
  sourceData: SourceData
): ParsedDraft {
  const state = parseInitialState(sourceData);
  const generator = new TimelineGenerator();
  const events = generator.generate(state, sourceData.events);

  return {
    name: sourceData.name,
    isComplete: generator.isComplete(),
    state,
    events,
  };
}

export interface ParsedDraft {
  name: string,
  isComplete: boolean,
  state: DraftState,
  events: TimelineEvent[],
}
