import { ExtensibleError } from '../util/ExtensibleError';

export class MutationError extends ExtensibleError {
  constructor(message: string) {
    super(message);
  }
}
