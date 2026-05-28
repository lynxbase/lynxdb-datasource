import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { QueryEditorHelp } from './components/QueryEditorHelp';
import { LynxQuery, LynxDataSourceOptions } from './types';

export const plugin = new DataSourcePlugin<DataSource, LynxQuery, LynxDataSourceOptions>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor)
  .setQueryEditorHelp(QueryEditorHelp);
