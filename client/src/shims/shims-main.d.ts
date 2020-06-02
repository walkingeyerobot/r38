export {};

declare global {
  export interface Window {
    DraftString?: string;
    UserInfo?: string;
  }

  const DEVELOPMENT: boolean;
}
