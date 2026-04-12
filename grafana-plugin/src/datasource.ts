import {
  DataSourceInstanceSettings,
  DataQueryRequest,
  DataQueryResponse,
  DataFrameView,
  MutableDataFrame,
  FieldType,
} from '@grafana/data';
import { DataSourceWithBackend, getBackendSrv } from '@grafana/runtime';

export interface OAuth4osQuery {
  refId: string;
  index: string;
  queryBody: string;
  queryType: 'search' | 'count' | 'cat';
}

export interface OAuth4osOptions {
  proxyUrl: string;
}

export class DataSource extends DataSourceWithBackend<OAuth4osQuery, OAuth4osOptions> {
  proxyUrl: string;

  constructor(instanceSettings: DataSourceInstanceSettings<OAuth4osOptions>) {
    super(instanceSettings);
    this.proxyUrl = instanceSettings.jsonData.proxyUrl || '';
  }

  async query(request: DataQueryRequest<OAuth4osQuery>): Promise<DataQueryResponse> {
    const frames = await Promise.all(
      request.targets.map(async (target) => {
        const frame = new MutableDataFrame({ refId: target.refId, fields: [] });

        try {
          const result = await this.doRequest(target);
          if (target.queryType === 'count') {
            frame.addField({ name: 'count', type: FieldType.number });
            frame.add({ count: result.count ?? 0 });
          } else if (target.queryType === 'cat') {
            // _cat/indices returns array of objects
            if (Array.isArray(result)) {
              const keys = Object.keys(result[0] || {});
              keys.forEach((k) => frame.addField({ name: k, type: FieldType.string }));
              result.forEach((row: Record<string, string>) => frame.add(row));
            }
          } else {
            // Search results
            const hits = result.hits?.hits || [];
            if (hits.length > 0) {
              const fields = Object.keys(hits[0]._source || {});
              fields.forEach((f) => frame.addField({ name: f, type: FieldType.string }));
              hits.forEach((hit: any) => frame.add(hit._source));
            }
          }
        } catch (err: any) {
          frame.addField({ name: 'error', type: FieldType.string });
          frame.add({ error: err.message || 'Query failed' });
        }

        return frame;
      })
    );

    return { data: frames };
  }

  private async doRequest(target: OAuth4osQuery): Promise<any> {
    const proxy = `api/datasources/proxy/${this.id}`;

    if (target.queryType === 'cat') {
      return getBackendSrv().get(`${proxy}/_cat/indices?format=json`);
    }

    if (target.queryType === 'count') {
      return getBackendSrv().post(`${proxy}/${target.index}/_count`, JSON.parse(target.queryBody || '{}'));
    }

    // Default: search
    return getBackendSrv().post(`${proxy}/${target.index}/_search`, JSON.parse(target.queryBody || '{"query":{"match_all":{}}}'));
  }

  async testDatasource(): Promise<{ status: string; message: string }> {
    try {
      const proxy = `api/datasources/proxy/${this.id}`;
      await getBackendSrv().get(`${proxy}/_cluster/health`);
      return { status: 'success', message: 'Connected to OpenSearch via oauth4os' };
    } catch (err: any) {
      return { status: 'error', message: err.message || 'Connection failed' };
    }
  }
}
