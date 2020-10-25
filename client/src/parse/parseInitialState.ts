import { CardContainer, CardPack, DraftCard, DraftSeat, DraftState, PlayerPicks, PackContainer, PACK_LOCATION_UNUSED, PACK_LOCATION_DEAD, MtgCard, CONTAINER_SHADOW } from '../draft/DraftState';
import { SourceCard, SourceData, SourceSeat } from './SourceData';
import { checkNotNil } from '../util/checkNotNil';
import DefaultAvatar from '../ui/shared/avatars/default_avatar.png';


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

    const shadowPool: PlayerPicks = {
      type: 'shadow-realm',
      id: CONTAINER_SHADOW,
      owningSeat: -1,
      cards: [],
      count: 0,
    };
    this._packs.set(shadowPool.id, shadowPool);

    const seats = [] as DraftSeat[];
    for (const [i, srcSeat] of srcData.seats.entries()) {
      seats.push(this.buildSeat(i, srcSeat));
    }

    return {
      state: {
        seats,
        shadowPool,
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
      type: 'seat',
      id: this._nextContainerId++,
      owningSeat: position,
      cards: [],
      count: 0,
    };
    this._packs.set(playerPicks.id, playerPicks);

    const seat: DraftSeat = {
      position: position,
      player: {
        id: src.playerId,
        name: src.playerName || 'Unknown player',
        mtgoName: src.mtgoName,
        iconUrl: src.playerImage || DefaultAvatar,
        seatPosition: position,
      },
      picks: playerPicks,
      originalPacks: [],
      queuedPacks:
          this.buildPackLocation(
              this._nextLocationId++,
              `queuedPacks for seat ${position}`),
      round: 1,
      colorCounts: { w: 0, u: 0, b: 0, r: 0, g: 0, },
    };

    this.parsePacks(seat, src);

    return seat;
  }

  private parsePacks(seat: DraftSeat, src: SourceSeat) {
    for (let i = 0; i < src.packs.length; i++) {
      const srcPack = src.packs[i];
      const cards = this.parseCards(srcPack);
      const pack: CardPack = {
        type: 'pack',
        id: this._nextContainerId++,
        round: i + 1,
        epoch: 0,
        cards: cards,
        count: cards.length,
        labelId: this._nextPackLabelId,
        originalSeat: seat.position,
      };
      seat.queuedPacks.packs.push(pack);
      seat.originalPacks.push(pack.id);
      this._packs.set(pack.id, pack);
    }
  }

  private parseCards(srcPack: SourceCard[]) {
    const cards = [] as number[];
    for (let i = 0; i < srcPack.length; i++) {
      const srcCard = srcPack[i];
      const card: DraftCard = {
        id: srcCard.id,
        sourcePackIndex: i,
        pickedIn: [],
        hidden: srcCard.hidden || false,
        // TODO: We should freeze the entire DraftCard, but pickedIn is still
        // mutable at the moment.
        definition: Object.freeze(parseCardDefinition(srcCard)),
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

function parseCardDefinition(src: SourceCard): MtgCard {
  if (src.hidden) {
    return {
      name: 'Hidden card',
      mana_cost: '',
      cmc: 0,
      collector_number: '-1',
      card_faces: [],
      colors: [],
      color_identity: [],
      foil: false,
      image_uris: [],
      layout: 'normal',
      mtgo: -1,
      rarity: 'common',
      set: '',
      type_line: '',
      searchName: '',
    };
  } else {
    return {
      name: src.scryfall.name,

      cmc: src.scryfall.cmc,
      collector_number: src.scryfall.collector_number,
      card_faces: (src.scryfall.card_faces || [])
          .map(face => ({
            name: face.name,
            colors: face.colors || [],
            mana_cost: face.mana_cost,
            type_line: face.type_line,
          })),
      color_identity: src.scryfall.color_identity,
      colors: src.scryfall.colors || [],
      foil: src.foil,
      image_uris: checkNotNil(src.image_uris),
      layout: src.scryfall.layout,
      mana_cost: src.scryfall.mana_cost || '',
      mtgo: src.mtgo_id,
      rarity: src.scryfall.rarity,
      set: src.scryfall.set,
      type_line: src.scryfall.type_line,

      searchName: src.scryfall.name.toLocaleLowerCase().normalize(),
    };
  }
}

interface StateParseResult {
  state: DraftState,
  cards: Map<number, DraftCard>,
}
