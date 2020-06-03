import { CardContainer, CardPack, DraftCard, DraftSeat, DraftState, PlayerPicks, PackContainer, PACK_LOCATION_UNUSED, PACK_LOCATION_DEAD } from '../draft/DraftState';
import { SourceCard, SourceData } from './SourceData';


export function parseInitialState(srcData: SourceData): DraftState {
  return new StateParser().parse(srcData);
}

class StateParser {
  private _nextLocationId = 0;
  private _nextPackId = 0;

  parse(srcData: SourceData): DraftState {
    const packMap = new Map<number, CardContainer>();
    const locationMap = new Map<number, PackContainer>();

    const unusedPacks =
        this.buildPackLocation(
            PACK_LOCATION_UNUSED,
            'Unused packs',
            locationMap);

    const deadPacks =
        this.buildPackLocation(
            PACK_LOCATION_DEAD,
            'Dead packs',
            locationMap);

    const state: DraftState = {
      seats: [],
      unusedPacks,
      deadPacks,
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
                `queuedPacks for seat ${i}`,
                locationMap),
        unopenedPacks:
            this.buildPackLocation(
                this._nextLocationId++,
                `unopenedPacks for seat ${i}`,
                locationMap),
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
    packMap.set(extraPack.id, extraPack);

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
      registrationMap: Map<number, PackContainer>,
  ): PackContainer {
    const packLocation = {
      id: id,
      packs: [],
      label: label,
    };
    registrationMap.set(id, packLocation);
    return packLocation;
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
