import { DraftState, CardContainer, PackContainer } from './DraftState';

export function cloneDraftState(src: DraftState) {
  const clonedState: DraftState = JSON.parse(JSON.stringify(src));

  clonedState.packs = new Map<number, CardContainer>();
  clonedState.locations = new Map<number, PackContainer>();

  registerLocation(clonedState, clonedState.unusedPacks);
  registerLocation(clonedState, clonedState.deadPacks);

  registerCardContainers(clonedState, clonedState.unusedPacks.packs);
  registerCardContainers(clonedState, clonedState.deadPacks.packs);
  for (const seat of clonedState.seats) {
    registerCardContainers(clonedState, [seat.player.picks]);
    registerCardContainers(clonedState, seat.queuedPacks.packs);
    registerCardContainers(clonedState, seat.unopenedPacks.packs);

    registerLocation(clonedState, seat.queuedPacks);
    registerLocation(clonedState, seat.unopenedPacks);
  }

  return clonedState;
}

function registerLocation(
    state: DraftState,
    location: PackContainer,
) {
  state.locations.set(location.id, location);
}

function registerCardContainers(
  state: DraftState,
  containers: CardContainer[]
) {
  for (const container of containers) {
    state.packs.set(container.id, container);
  }
}
