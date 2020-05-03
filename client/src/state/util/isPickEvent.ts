import { TimelineEvent, ActionMoveCard } from '../../draft/TimelineEvent';
import { find } from '../../util/collection';

export function isPickEvent(event: TimelineEvent): boolean {
  return find(event.actions, { type: 'move-card' }) != -1;
}

export function getPickAction(event: TimelineEvent): ActionMoveCard | null {
  const index = find(event.actions, { type: 'move-card' });
  if (index != -1) {
    return event.actions[index] as ActionMoveCard;
  } else {
    return null;
  }
}
