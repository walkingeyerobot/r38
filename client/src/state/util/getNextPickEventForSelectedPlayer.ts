import { CoreState } from '../store';
import { TimelineEvent } from '../../draft/TimelineEvent';
import { isPickEvent } from './isPickEvent';

export function getNextPickEventForSelectedPlayer(
  state: CoreState,
): TimelineEvent | null {
  const seatId = state.selection?.id;
  let pickEvent: TimelineEvent | null = null;
  for (let i = state.eventPos; i < state.events.length; i++) {
    const event = state.events[i];
    if (event.associatedSeat == seatId && isPickEvent(event)) {
      pickEvent = event;
      break;
    }
  }
  return pickEvent;
}
