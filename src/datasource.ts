import {
  CoreApp,
  DataQueryResponse,
  DataQueryRequest,
  DataSourceInstanceSettings,
  LiveChannelScope,
  MetricFindValue,
  ScopedVars,
  SupplementaryQueryOptions,
  SupplementaryQueryType,
} from '@grafana/data';
import { DataSourceWithBackend, getGrafanaLiveSrv, getTemplateSrv } from '@grafana/runtime';
import { Observable, merge } from 'rxjs';

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

  // query routes Explore live-tail requests to a Grafana Live stream backed by
  // the plugin RunStream handler; everything else uses the standard backend query.
  query(request: DataQueryRequest<LynxQuery>): Observable<DataQueryResponse> {
    if (request.liveStreaming) {
      return this.tailQuery(request);
    }
    return super.query(request);
  }

  private tailQuery(request: DataQueryRequest<LynxQuery>): Observable<DataQueryResponse> {
    const live = getGrafanaLiveSrv();
    if (!live) {
      return super.query(request);
    }

    const streams = request.targets
      .filter((target) => this.filterQuery(target))
      .map((target) => {
        const q = this.applyTemplateVariables(target, request.scopedVars);
        return live.getDataStream({
          addr: {
            scope: LiveChannelScope.DataSource,
            stream: this.uid,
            path: `tail/${target.refId}/${hashQuery(q.queryText)}`,
            data: { queryText: q.queryText, queryType: q.queryType, maxLines: q.maxLines },
          },
        });
      });

    return streams.length > 0 ? merge(...streams) : super.query(request);
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

  // --- Template variables --------------------------------------------------

  // metricFindQuery powers query variables. Supported queries:
  //   fields           -> all field names
  //   sources          -> all source names
  //   values(<field>)  -> distinct values of a field
  // A bare field name is treated as values(<field>).
  async metricFindQuery(query: string): Promise<MetricFindValue[]> {
    const raw = getTemplateSrv().replace(query ?? '').trim();

    if (raw === '' || raw === 'fields') {
      const fields = await this.getFields();
      return fields.map((f) => ({ text: f.name }));
    }
    if (raw === 'sources') {
      const sources = await this.getSources();
      return sources.map((s) => ({ text: s.name }));
    }

    const valuesMatch = raw.match(/^values\(([^)]+)\)$/);
    const field = valuesMatch ? valuesMatch[1].trim() : raw;
    const values = await this.getFieldValues(field, 1000);
    return values.map((v) => ({ text: String(v.value) }));
  }
}

// hashQuery produces a short channel-path-safe token so distinct queries on the
// same refId get distinct live channels.
function hashQuery(text: string): string {
  let hash = 5381;
  for (let i = 0; i < text.length; i++) {
    hash = (hash * 33 + text.charCodeAt(i)) | 0;
  }
  return (hash >>> 0).toString(36);
}
