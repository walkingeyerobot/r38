import { SourceData } from './SourceData';
import { parseInitialState } from './parseInitialState';
import { TimelineGenerator } from './TimelineGenerator';
import { DraftState, TimelineEvent } from '../draft/draft_types';

export function parseDraft(
  sourceData: SourceData
): { state: DraftState, events: TimelineEvent[] } {
  const state = parseInitialState(sourceData);
  const events = new TimelineGenerator().generate(state, sourceData.events);
  return { state, events };
}
