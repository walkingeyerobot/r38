import { TimelineEvent } from '../../draft/TimelineEvent';
import { find } from '../../util/collection';

export function isPickEvent(event: TimelineEvent): boolean {
  return find(event.actions, { type: 'move-card' }) != -1;
}
