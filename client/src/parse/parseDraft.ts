import { SourceData } from './SourceData';
import { parseInitialState } from './parseInitialState';
import { TimelineGenerator } from './TimelineGenerator';
import { DraftState } from '../draft/DraftState';
import { TimelineEvent } from '../draft/TimelineEvent';

export function parseDraft(
  sourceData: SourceData
): { state: DraftState, events: TimelineEvent[] } {
  const state = parseInitialState(sourceData);
  const events = new TimelineGenerator().generate(state, sourceData.events);
  return { state, events };
}
