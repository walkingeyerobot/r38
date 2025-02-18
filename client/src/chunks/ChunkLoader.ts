// TODO: Figure out an elegant way to proxy all properties of the target
// chunk, so we can get rid of the `chunk` property.
export class ChunkLoader<T> {
  private _chunk: T | null = null;
  private _chunkPromise: Promise<T> | null = null;

  constructor(private _importer: () => Promise<T>) {}

  get chunk(): T {
    if (this._chunk == null) {
      throw new Error(`Chunk not yet initialized!`);
    }
    return this._chunk;
  }

  get loaded(): boolean {
    return this._chunk != null;
  }

  load(): Promise<T> {
    if (this._chunkPromise == null) {
      this._chunkPromise = this._importer().then((chunk) => {
        this._chunk = chunk;
        return chunk;
      });
    }
    return this._chunkPromise;
  }
}
