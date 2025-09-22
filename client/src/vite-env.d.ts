/// <reference types="vite/client" />

interface ViteTypeOptions {
  // By adding this line, you can make the type of ImportMetaEnv strict
  // to disallow unknown keys.
  strictImportMetaEnv: unknown;
}

interface ImportMetaEnv {
  readonly VITE_APP_TITLE: string;
  readonly VITE_DISABLE_STATE_POLLING: boolean | undefined;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
