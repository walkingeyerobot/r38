import { CardContainer, CardPack, DraftCard, DraftSeat, DraftState, PlayerPicks, PackContainer, PACK_LOCATION_UNUSED, PACK_LOCATION_DEAD } from '../draft/DraftState';
import { SourceCard, SourceData, SourceSeat } from './SourceData';


export function parseInitialState(srcData: SourceData): StateParseResult {
  return new StateParser().parse(srcData);
}

class StateParser {
  private _nextLocationId = 0;
  private _nextContainerId = 0;
  private _nextPackLabelId = 0;
  private _packs = new Map<number, CardContainer>();
  private _locations = new Map<number, PackContainer>();
  private _cards = new Map<number, DraftCard>();

  parse(srcData: SourceData): StateParseResult {
    const unusedPacks =
        this.buildPackLocation(PACK_LOCATION_UNUSED, 'Unused packs');

    const deadPacks =
        this.buildPackLocation(PACK_LOCATION_DEAD, 'Dead packs');

    const seats = [] as DraftSeat[];
    for (const [i, srcSeat] of srcData.seats.entries()) {
      seats.push(this.buildSeat(i, srcSeat));
    }

    const shadowSeat = this.buildSeat(-1, {
      playerId: -1,
      playerName: 'The Shadow',
      playerImage: 'SHADOW',
      packs: [],
    });

    return {
      state: {
        seats,
        shadowSeat,
        unusedPacks,
        deadPacks,
        packs: this._packs,
        locations: this._locations
      },
      cards: this._cards,
    };
  }

  private buildSeat(position: number, src: SourceSeat) {
    const playerPicks: PlayerPicks = {
      id: this._nextContainerId++,
      type: 'seat',
      cards: []
    };
    this._packs.set(playerPicks.id, playerPicks);

    const seat: DraftSeat = {
      position: position,
      player: {
        id: src.playerId,
        name: src.playerName || 'Unknown player',
        iconUrl: src.playerImage,
        seatPosition: position,
        picks: playerPicks,
      },
      originalPacks: [],
      queuedPacks:
          this.buildPackLocation(
              this._nextLocationId++,
              `queuedPacks for seat ${position}`),
      unopenedPacks:
          this.buildPackLocation(
              this._nextLocationId++,
              `unopenedPacks for seat ${position}`),
    };

    this.parsePacks(seat, src);

    return seat;
  }

  private parsePacks(seat: DraftSeat, src: SourceSeat) {
    for (let i = 0; i < src.packs.length; i++) {
      const srcPack = src.packs[i];
      const pack: CardPack = {
        type: 'pack',
        id: this._nextContainerId++,
        round: i + 1,
        cards: this.parseCards(srcPack),
        labelId: this._nextPackLabelId,
        originalSeat: seat.position,
      };
      seat.unopenedPacks.packs.push(pack);
      seat.originalPacks.push(pack.id);
      this._packs.set(pack.id, pack);
    }
  }

  private parseCards(srcPack: SourceCard[]) {
    const cards = [] as number[];
    for (let i = 0; i < srcPack.length; i++) {
      const srcCard = srcPack[i];
      const card: DraftCard = {
        id: srcCard.r38_data.id,
        sourcePackIndex: i,
        pickedIn: [],

        // TODO: We should freeze the entire DraftCard, but pickedIn is still
        // mutable at the moment.
        definition: Object.freeze({
          name: srcCard.name,
          set: srcCard.set,
          collector_number: srcCard.collector_number,
          mana_cost: srcCard.mana_cost || '',
          cmc: srcCard.cmc,
          colors: srcCard.colors || [],
          color_identity: srcCard.color_identity,
          rarity: srcCard.rarity,
          type_line: srcCard.type_line,
          layout: srcCard.layout,
          card_faces: (srcCard.card_faces || [])
              .map(face => ({
                name: face.name,
                colors: face.colors || [],
                mana_cost: face.mana_cost,
                type_line: face.type_line,
              })),

          mtgo: srcCard.r38_data.mtgo_id,
          foil: srcCard.r38_data.foil,
          searchName: srcCard.name.toLocaleLowerCase().normalize(),
        }),
      };
      if (this._cards.has(card.id)) {
        throw new Error(`Duplicate card ${card.id}`);
      }
      this._cards.set(card.id, card);

      cards.push(card.id);
    }

    return cards;
  }

  private buildPackLocation(
      id: number,
      label: string,
  ): PackContainer {
    const packLocation = {
      id: id,
      packs: [],
      label: label,
    };
    this._locations.set(id, packLocation);
    return packLocation;
  }
}

interface StateParseResult {
  state: DraftState,
  cards: Map<number, DraftCard>,
}
