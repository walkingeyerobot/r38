import { TimelineEvent } from '../../draft/TimelineEvent';
import { isPickEvent } from './isPickEvent';
import { ReplayModule } from '../ReplayModule';

export function getNextPickEventForSelectedPlayer(
  store: ReplayModule,
): TimelineEvent | null {
  const seatId = store.selection?.id;
  let pickEvent: TimelineEvent | null = null;
  for (let i = store.eventPos; i < store.events.length; i++) {
    const event = store.events[i];
    if (event.associatedSeat == seatId && isPickEvent(event)) {
      pickEvent = event;
      break;
    }
  }
  return pickEvent;
}

export function getNextPickEvent(
  store: ReplayModule,
): TimelineEvent | null {
  let pickEvent: TimelineEvent | null = null;
  for (let i = store.eventPos; i < store.events.length; i++) {
    const event = store.events[i];
    if (isPickEvent(event)) {
      pickEvent = event;
      break;
    }
  }
  return pickEvent;
}
