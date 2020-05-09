import { FAKE_DATA_03 } from './FAKE_DATA_03';
import { SourceData } from '../parse/SourceData';

/**
 * Precanned data for use during local development
 *
 * This file is replaced bt devReplayData.stub.ts for production builds (see
 * /build_config/webpack.prod.js).
 */
export const devReplayData: SourceData | null = FAKE_DATA_03;
