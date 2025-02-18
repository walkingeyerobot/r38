import { SourceData } from "@/parse/SourceData";

export {};

declare global {
  export interface Window {
    UserInfo?: string;
    draftData?: SourceData;
  }

  const DEVELOPMENT: boolean;
}
