import { DraftState, DraftCard, DraftSeat, CardPack, CardContainer, PlayerPicks } from '../draft/draft_types';
import { SourceData, SourceCard } from './SourceData';

// TODO: Make this something more than just a static global
let nextCardId = 0;

export function parseInitialState(srcData: SourceData): DraftState {
  let nextPackId = 0;
  const packMap = new Map<number, CardContainer>();

  const state: DraftState = {
    seats: [],
    unusedPacks: [],
    packs: packMap,
  };

  for (const [i, srcSeat] of srcData.seats.entries()) {
    const playerPicks: PlayerPicks = {
      id: nextPackId++,
      type: 'player-picks',
      cards: []
    };
    packMap.set(playerPicks.id, playerPicks);

    const seat: DraftSeat = {
      position: i,
      player: {
        name: srcSeat.name || FAKE_PLAYER_NAMES[i],
        seatPosition: i,
        picks: playerPicks,
      },
      originalPacks: [],
      queuedPacks: [],
      unopenedPacks: [],
    };
    state.seats.push(seat);
  }

  for (const [i, srcSeat] of srcData.seats.entries()) {
    const seat = state.seats[i];
    for (let j = 1; j < srcSeat.rounds.length; j++) {
      const pack: CardPack = {
        id: nextPackId++,
        type: 'pack',
        cards: parsePack(srcSeat.rounds[j].packs[0].cards),
        originalSeat: i
      };
      seat.unopenedPacks.push(pack);
      seat.originalPacks.push(pack.id);
      packMap.set(pack.id, pack);
    }
  }

  // Add the extra pack to the unused packs area
  const extraPack: CardPack = {
    id: nextPackId++,
    type: 'pack',
    cards: parsePack(srcData.extraPack || []),
    originalSeat: -1,
  };
  state.unusedPacks.push(extraPack);
  state.packs.set(extraPack.id, extraPack);

  return state;
}

function parsePack(srcPack: SourceCard[]) {
  const pack = [] as DraftCard[];
  for (let i = 0; i < srcPack.length; i++) {
    const srcPick = srcPack[i];
    pack.push({
      id: nextCardId++,
      definition: {
        name: srcPick.name,
        set: srcPick.edition,
        collector_number: srcPick.number,
        tags: srcPick.tags.split(", "),
      },
      sourcePackIndex: i,
      draftedBy: null,
    });
  }

  return pack;
}

const FAKE_PLAYER_NAMES = [
  'Tamanna',
  'Anna-Marie',
  'Abbigail',
  'Riley',
  'Matas',
  'Clive',
  'Axl',
  'Isobel',
];