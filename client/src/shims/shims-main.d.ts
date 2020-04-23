export {};

declare global {
  export interface Window {
    DraftString?: string;
  }

  const DEVELOPMENT: boolean;
}
