import { ReplayStore } from "../ReplayStore";
import { isPickEvent } from './isPickEvent';
import { nil } from '../../util/nil';

export function getNextPickEvent(store: ReplayStore, seatId: number | nil) {
  for (let i = store.eventPos; i < store.events.length; i++) {
    const event = store.events[i];
    if (isPickEvent(event)
        && (seatId == undefined || event.associatedSeat == seatId)) {
      return event;
    }
  }
  return null;
}

export function getPreviousPickEvent(store: ReplayStore, seatId: number | nil) {
  for (let i = store.eventPos - 1; i >= 0; i--) {
    const event = store.events[i];
    if (isPickEvent(event)
        && (seatId == undefined || event.associatedSeat == seatId)) {
      return event;
    }
  }
  return null;
}
