import { ExtensibleError } from '../util/ExtensibleError';

export class ParseError extends ExtensibleError {
  constructor(message: string) {
    super(message);
  }
}
