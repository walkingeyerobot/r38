import { SourceData } from '../parse/SourceData';
import { stubDraft_17 } from '../rest/api/draft/draft.17.stub';

/**
 * Precanned data for use during local development
 *
 * This file is replaced bt devReplayData.stub.ts for production builds (see
 * /build_config/webpack.prod.js).
 */
// TODO: Remove this once we switch over to all-REST endpoints
export const devReplayData: SourceData | null = stubDraft_17;
