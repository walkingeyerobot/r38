import { SourceData } from '../parse/SourceData';
import { DRAFT_17 } from './DRAFT_17';

/**
 * Precanned data for use during local development
 *
 * This file is replaced bt devReplayData.stub.ts for production builds (see
 * /build_config/webpack.prod.js).
 */
export const devReplayData: SourceData | null = DRAFT_17;
