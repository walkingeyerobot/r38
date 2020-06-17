import { TimelineEvent, ActionMoveCard } from '../../draft/TimelineEvent';
import { indexOf } from '../../util/collection';

export function isPickEvent(event: TimelineEvent): boolean {
  return indexOf(event.actions, { type: 'move-card' }) != -1;
}

export function getPickAction(event: TimelineEvent): ActionMoveCard | null {
  const index = indexOf(event.actions, { subtype: 'pick-card' });
  if (index != -1) {
    return event.actions[index] as ActionMoveCard;
  } else {
    return null;
  }
}
