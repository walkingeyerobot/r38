import { TimelineEvent } from '../../draft/TimelineEvent';
import { DraftState } from '../../draft/DraftState';

export function printEvent(event: TimelineEvent, state: DraftState) {
  console.log(
      'EVENT', event.id,
      `seat=${event.associatedSeat}`,
      `round=${event.round}`,
      `roundEpoch=${event.roundEpoch}`
  );
  for (let action of event.actions) {
    switch (action.type) {
      case 'move-pack':
        console.log(
            '  ', action.type,
            `pack=${action.pack}`,
            `from=${action.from}`,
            `"${state.locations.get(action.from)?.label}"`,
            `to=${action.to}`,
            `"${state.locations.get(action.to)?.label}"`
            );
        break;
      case 'move-card':
        console.log(
          '  ', action.type,
          `card="${action.cardName}"`,
          `from=${action.from}`,
          `to=${action.to}`
          );
    }
  }
}
