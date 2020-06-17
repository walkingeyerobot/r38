import { DraftCard } from './DraftState';

export interface CardStore {
  getCard(id: number): DraftCard;
}
