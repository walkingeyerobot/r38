import { CardContainer, CardPack, DraftCard, DraftSeat, DraftState, PlayerPicks, PackContainer, PACK_LOCATION_UNUSED, PACK_LOCATION_DEAD } from '../draft/DraftState';
import { SourceCard, SourceData } from './SourceData';

// TODO: Make this something more than just a static global
let nextCardId = 0;
let nextLocationId = 0;

export function parseInitialState(srcData: SourceData): DraftState {
  let nextPackId = 0;
  const packMap = new Map<number, CardContainer>();
  const locationMap = new Map<number, PackContainer>();

  const state: DraftState = {
    seats: [],
    unusedPacks: buildPackLocation(PACK_LOCATION_UNUSED, 'Unused packs'),
    deadPacks: buildPackLocation(PACK_LOCATION_DEAD, 'Dead packs'),
    packs: packMap,
    locations: locationMap,
  };

  for (const [i, srcSeat] of srcData.seats.entries()) {
    const playerPicks: PlayerPicks = {
      id: nextPackId++,
      type: 'seat',
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
      queuedPacks:
          buildPackLocation(nextLocationId++, `queuedPacks for seat ${i}`),
      unopenedPacks:
          buildPackLocation(nextLocationId++, `unopenedPacks for seat ${i}`),
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
        originalSeat: i,
        round: j,
      };
      seat.unopenedPacks.packs.push(pack);
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
    round: -1,
  };
  state.unusedPacks.packs.push(extraPack);
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
        cmc: srcPick.cmc,
        color: srcPick.color,
        mtgo: srcPick.mtgo || "",
        tags: srcPick.tags.split(", "),
        searchName: srcPick.name.toLocaleLowerCase().normalize(),
      },
      sourcePackIndex: i,
      pickedIn: [],
    });
  }

  return pack;
}

function buildPackLocation(id: number, label: string): PackContainer {
  return {
    id: id,
    packs: [],
    label: label,
  };
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
