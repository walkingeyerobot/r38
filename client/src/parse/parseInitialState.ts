import { CardContainer, CardPack, DraftCard, DraftSeat, DraftState, PlayerPicks, PackContainer, PACK_LOCATION_UNUSED, PACK_LOCATION_DEAD } from '../draft/DraftState';
import { SourceCard, SourceData } from './SourceData';
import { fillDraftStateMaps } from './fillDraftStateMaps';


export function parseInitialState(srcData: SourceData): DraftState {
  const state = new StateParser().parse(srcData);

  fillDraftStateMaps(state);

  return state;
}

class StateParser {
  private _nextLocationId = 0;
  private _nextPackId = 0;

  parse(srcData: SourceData): DraftState {
    const packMap = new Map<number, CardContainer>();
    const locationMap = new Map<number, PackContainer>();

    const state: DraftState = {
      seats: [],
      unusedPacks: this.buildPackLocation(PACK_LOCATION_UNUSED, 'Unused packs'),
      deadPacks: this.buildPackLocation(PACK_LOCATION_DEAD, 'Dead packs'),
      packs: packMap,
      locations: locationMap,
    };

    for (const [i, srcSeat] of srcData.seats.entries()) {
      const playerPicks: PlayerPicks = {
        id: this._nextPackId++,
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
            this.buildPackLocation(
                this._nextLocationId++,
                `queuedPacks for seat ${i}`),
        unopenedPacks:
            this.buildPackLocation(
                this._nextLocationId++,
                `unopenedPacks for seat ${i}`),
      };
      state.seats.push(seat);
    }

    for (const [i, srcSeat] of srcData.seats.entries()) {
      const seat = state.seats[i];
      for (let j = 1; j < srcSeat.rounds.length; j++) {
        const pack: CardPack = {
          id: this._nextPackId++,
          type: 'pack',
          cards: this.parsePack(srcSeat.rounds[j].packs[0].cards),
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
      id: this._nextPackId++,
      type: 'pack',
      cards: this.parsePack(srcData.extraPack || []),
      originalSeat: -1,
      round: -1,
    };
    state.unusedPacks.packs.push(extraPack);
    state.packs.set(extraPack.id, extraPack);

    return state;
  }

  private parsePack(srcPack: SourceCard[]) {
    const pack = [] as DraftCard[];
    for (let i = 0; i < srcPack.length; i++) {
      const srcPick = srcPack[i];
      pack.push({
        id: srcPick.r38Id,
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

  private buildPackLocation(
      id: number,
      label: string,
  ): PackContainer {
    return {
      id: id,
      packs: [],
      label: label,
    };
  }
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
