import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './config';
import { QueryEditor } from './query';

export const plugin = new DataSourcePlugin(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);
