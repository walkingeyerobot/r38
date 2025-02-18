export type SelectedView = SelectedPack | SelectedPlayer;

export interface SelectedPack {
  type: "pack";
  id: number;
}

export interface SelectedPlayer {
  type: "seat";
  id: number;
}
