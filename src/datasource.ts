import {
  CoreApp,
  DataQueryRequest,
  DataSourceInstanceSettings,
  ScopedVars,
  SupplementaryQueryOptions,
  SupplementaryQueryType,
} from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import {
  DEFAULT_QUERY,
  ExplainResult,
  FieldInfo,
  FieldValue,
  LOG_VOLUME_QUERY_TYPE,
  LynxDataSourceOptions,
  LynxQuery,
  SourceInfo,
} from './types';

export class DataSource extends DataSourceWithBackend<LynxQuery, LynxDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<LynxDataSourceOptions>) {
    super(instanceSettings);
  }

  getDefaultQuery(_: CoreApp): Partial<LynxQuery> {
    return DEFAULT_QUERY;
  }

  applyTemplateVariables(query: LynxQuery, scopedVars: ScopedVars): LynxQuery {
    return {
      ...query,
      queryText: query.queryText ? getTemplateSrv().replace(query.queryText, scopedVars) : query.queryText,
    };
  }

  filterQuery(query: LynxQuery): boolean {
    return Boolean(query.queryText && query.queryText.trim() !== '');
  }

  // --- Explore log-volume support -----------------------------------------

  getSupportedSupplementaryQueryTypes(): SupplementaryQueryType[] {
    return [SupplementaryQueryType.LogsVolume];
  }

  getSupplementaryQuery(options: SupplementaryQueryOptions, query: LynxQuery): LynxQuery | undefined {
    if (options.type !== SupplementaryQueryType.LogsVolume || !query.queryText) {
      return undefined;
    }
    return {
      ...query,
      refId: `${LOG_VOLUME_QUERY_TYPE}-${query.refId}`,
      queryType: LOG_VOLUME_QUERY_TYPE,
    } as unknown as LynxQuery;
  }

  getSupplementaryRequest(
    type: SupplementaryQueryType,
    request: DataQueryRequest<LynxQuery>
  ): DataQueryRequest<LynxQuery> | undefined {
    if (type !== SupplementaryQueryType.LogsVolume) {
      return undefined;
    }
    const targets = request.targets
      .map((q) => this.getSupplementaryQuery({ type }, q))
      .filter((q): q is LynxQuery => Boolean(q));
    if (targets.length === 0) {
      return undefined;
    }
    return { ...request, targets };
  }

  // --- Resource calls (autocomplete & validation) --------------------------

  async getFields(prefix?: string): Promise<FieldInfo[]> {
    const res = await this.getResource('fields', prefix ? { prefix } : undefined);
    return res?.fields ?? [];
  }

  async getFieldValues(field: string, limit = 50): Promise<FieldValue[]> {
    const res = await this.getResource('field-values', { field, limit });
    return res?.values ?? [];
  }

  async getSources(pattern?: string): Promise<SourceInfo[]> {
    const res = await this.getResource('sources', pattern ? { pattern } : undefined);
    return res?.sources ?? [];
  }

  async explain(queryText: string): Promise<ExplainResult> {
    return this.getResource('explain', { q: queryText });
  }
}
