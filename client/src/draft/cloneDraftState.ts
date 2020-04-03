import { DraftState, CardPack, CardContainer } from './draft_types';

export function cloneDraftState(src: DraftState) {
  const clonedState: DraftState = JSON.parse(JSON.stringify(src));

  const containers = [] as CardContainer[];
  containers.push(...clonedState.unusedPacks);
  for (const seat of clonedState.seats) {
    containers.push(seat.player.picks);
    containers.push(...seat.queuedPacks);
    containers.push(...seat.unopenedPacks);
  }

  containers.sort((a, b) => a.id - b.id);

  const newMap = new Map<number, CardContainer>();
  for (const container of containers) {
    newMap.set(container.id, container);
  }

  clonedState.packs = newMap;

  return clonedState;
}
