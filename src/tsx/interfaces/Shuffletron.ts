export interface STList {
  id: number;
  name: string;
}

export interface STGame {
  id: number;
  listId: number;
  name: string;
  displayName: string;
  description: string;
  weight: number;
  status: number;
}

export interface STShuffleResult {
  game: STGame;
  animContent: string[];
}