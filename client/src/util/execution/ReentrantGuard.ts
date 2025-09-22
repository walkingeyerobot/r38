import { ref } from "vue";

export class ReentrantGuard<T> {
  private readonly storeError: boolean;

  private readonly _promise = ref<Promise<T> | null>(null);
  private readonly _error = ref<unknown | null>(null);

  constructor(args?: { storeError: boolean }) {
    this.storeError = args?.storeError ?? false;
  }

  get promise(): Promise<T> | null {
    return this._promise.value;
  }

  get error(): unknown {
    return this._error.value;
  }

  get isRunning(): boolean {
    return this._promise.value != null;
  }

  /**
   * Executes [lambda] and returns the resulting Promise. Subsequent calls to this method before
   * the Promise fulfills will skip executing [lambda] and return the original promise.
   */
  runExclusive(lambda: () => Promise<T>, force?: boolean) {
    if (this._promise.value && !force) {
      return this._promise;
    }
    if (this._error) {
      this._error.value = null;
    }
    try {
      this._promise.value = lambda();
    } catch (e) {
      if (this.storeError) {
        this._error.value = e;
      }
      throw e;
    }
    this._promise.value.then(
      (value: T) => {
        this._promise.value = null;
        return value;
      },
      (e) => {
        this._promise.value = null;
        if (this.storeError) {
          this._error.value = e;
        }
      },
    );
    return this._promise.value;
  }
}
