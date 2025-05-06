/// <reference types="vite/client" />

// The latest version of vuex messed up their package.json definition in a way
// that breaks their typing exports (how did no one catch this?). So we have
// to manually export their typings here.
declare module "vuex" {
  export * from "vuex/types/index.d.ts";
  export * from "vuex/types/helpers.d.ts";
  export * from "vuex/types/logger.d.ts";
  export * from "vuex/types/vue.d.ts";
}

// Declare the special RFID event that our enclosing apps will inject if
// someone scans a card
interface HTMLElementEventMap {
  rfidScan: CustomEvent<string>;
}

// RFID stuff
// Pasted from https://github.com/w3c/web-nfc/blob/gh-pages/web-nfc.d.ts

interface Window {
  NDEFMessage: NDEFMessage;
}
declare class NDEFMessage {
  constructor(messageInit: NDEFMessageInit);
  records: ReadonlyArray<NDEFRecord>;
}
declare interface NDEFMessageInit {
  records: NDEFRecordInit[];
}

declare type NDEFRecordDataSource = string | BufferSource | NDEFMessageInit;

interface Window {
  NDEFRecord: NDEFRecord;
}
declare class NDEFRecord {
  constructor(recordInit: NDEFRecordInit);
  readonly recordType: string;
  readonly mediaType?: string;
  readonly id?: string;
  readonly data?: DataView;
  readonly encoding?: string;
  readonly lang?: string;
  toRecords?: () => NDEFRecord[];
}
declare interface NDEFRecordInit {
  recordType: string;
  mediaType?: string;
  id?: string;
  encoding?: string;
  lang?: string;
  data?: NDEFRecordDataSource;
}

declare type NDEFMessageSource = string | BufferSource | NDEFMessageInit;

interface Window {
  NDEFReader: NDEFReader;
}
declare class NDEFReader extends EventTarget {
  constructor();
  onreading: (this: this, event: NDEFReadingEvent) => unknown;
  onreadingerror: (this: this, error: Event) => unknown;
  scan: (options?: NDEFScanOptions) => Promise<void>;
  write: (message: NDEFMessageSource, options?: NDEFWriteOptions) => Promise<void>;
}

interface Window {
  NDEFReadingEvent: NDEFReadingEvent;
}
declare class NDEFReadingEvent extends Event {
  constructor(type: string, readingEventInitDict: NDEFReadingEventInit);
  serialNumber: string;
  message: NDEFMessage;
}
interface NDEFReadingEventInit extends EventInit {
  serialNumber?: string;
  message: NDEFMessageInit;
}

interface NDEFWriteOptions {
  overwrite?: boolean;
  signal?: AbortSignal;
}
interface NDEFScanOptions {
  signal: AbortSignal;
}

type PermissionName = PermissionName | "nfc";
