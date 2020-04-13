import { SourceData } from './SourceData';
import { parseInitialState } from './parseInitialState';
import { TimelineGenerator } from './TimelineGenerator';
import { DraftState } from '../draft/DraftState';
import { TimelineEvent } from '../draft/TimelineEvent';

export function parseDraft(
  sourceData: SourceData
): { state: DraftState, events: TimelineEvent[] } {
  const state = parseInitialState(sourceData);
  const generator = new TimelineGenerator();
  const events = generator.generate(state, sourceData.events);
  // HACK: This should probably just be an event at the end of the event stream
  state.isComplete = generator.isComplete();
  return { state, events };
}
