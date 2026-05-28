import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

/** logs returns events (log lines); metrics returns timechart/aggregate results. */
export type LynxQueryType = 'logs' | 'metrics';

/** Internal query type used for the Explore log-volume supplementary query. */
export const LOG_VOLUME_QUERY_TYPE = 'logvolume';

export interface LynxQuery extends DataQuery {
  queryText: string;
  queryType: LynxQueryType;
  maxLines?: number;
}

export const DEFAULT_QUERY: Partial<LynxQuery> = {
  queryText: '',
  queryType: 'logs',
};

/** Options configured for each datasource instance (non-secret). */
export interface LynxDataSourceOptions extends DataSourceJsonData {
  defaultIndex?: string;
  timeField?: string;
  messageField?: string;
  levelField?: string;
  maxLines?: number;
}

/** Secret values stored encrypted and only readable by the backend. */
export interface LynxSecureJsonData {
  token?: string;
}

/** Shapes returned by the backend resource endpoints (autocomplete/validation). */
export interface FieldInfo {
  name: string;
  type?: string;
  cardinality?: number;
}

export interface FieldValue {
  value: unknown;
  count?: number;
  percent?: number;
}

export interface SourceInfo {
  name: string;
  event_count?: number;
}

export interface ExplainError {
  position?: number;
  length?: number;
  message: string;
  suggestion?: string;
}

export interface ExplainResult {
  is_valid: boolean;
  errors?: ExplainError[];
}
