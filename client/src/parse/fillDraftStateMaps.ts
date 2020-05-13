import { DraftState, PackContainer, CardContainer } from '../draft/DraftState';

export function fillDraftStateMaps(state: DraftState) {
  state.locations = new Map();
  state.packs = new Map();

  registerLocation(state, state.unusedPacks);
  registerLocation(state, state.deadPacks);

  for (const seat of state.seats) {
    registerCardContainers(state, [seat.player.picks]);
    registerLocation(state, seat.queuedPacks);
    registerLocation(state, seat.unopenedPacks);
  }
}

function registerLocation(
    state: DraftState,
    location: PackContainer,
) {
  state.locations.set(location.id, location);
  registerCardContainers(state, location.packs);
}

function registerCardContainers(
  state: DraftState,
  containers: CardContainer[]
) {
  for (const container of containers) {
    state.packs.set(container.id, container);
  }
}
