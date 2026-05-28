import React, { ChangeEvent } from 'react';
import { InlineField, Input, SecretInput, Stack } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { LynxDataSourceOptions, LynxSecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<LynxDataSourceOptions, LynxSecureJsonData> {}

const LABEL_WIDTH = 20;
const INPUT_WIDTH = 40;

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { jsonData, secureJsonFields, secureJsonData } = options;

  const onURLChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({ ...options, url: event.target.value });
  };

  const onJsonChange = (key: keyof LynxDataSourceOptions) => (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({ ...options, jsonData: { ...jsonData, [key]: event.target.value } });
  };

  const onMaxLinesChange = (event: ChangeEvent<HTMLInputElement>) => {
    const raw = event.target.value;
    onOptionsChange({ ...options, jsonData: { ...jsonData, maxLines: raw === '' ? undefined : Number(raw) } });
  };

  const onTokenChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({ ...options, secureJsonData: { ...secureJsonData, token: event.target.value } });
  };

  const onResetToken = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: { ...secureJsonFields, token: false },
      secureJsonData: { ...secureJsonData, token: '' },
    });
  };

  return (
    <Stack direction="column" gap={2}>
      <InlineField label="URL" labelWidth={LABEL_WIDTH} interactive tooltip="LynxDB base URL, e.g. http://localhost:3100">
        <Input
          id="config-editor-url"
          width={INPUT_WIDTH}
          value={options.url || ''}
          placeholder="http://localhost:3100"
          onChange={onURLChange}
        />
      </InlineField>

      <InlineField label="API token" labelWidth={LABEL_WIDTH} interactive tooltip="Bearer token sent to LynxDB. Stored encrypted.">
        <SecretInput
          id="config-editor-token"
          width={INPUT_WIDTH}
          isConfigured={Boolean(secureJsonFields?.token)}
          value={secureJsonData?.token || ''}
          placeholder="LynxDB API token"
          onReset={onResetToken}
          onChange={onTokenChange}
        />
      </InlineField>

      <InlineField label="Default index" labelWidth={LABEL_WIDTH} interactive tooltip="Optional default index/namespace">
        <Input
          id="config-editor-index"
          width={INPUT_WIDTH}
          value={jsonData.defaultIndex || ''}
          placeholder="main"
          onChange={onJsonChange('defaultIndex')}
        />
      </InlineField>

      <InlineField label="Time field" labelWidth={LABEL_WIDTH} interactive tooltip="Event timestamp field">
        <Input
          id="config-editor-time-field"
          width={INPUT_WIDTH}
          value={jsonData.timeField || ''}
          placeholder="_time"
          onChange={onJsonChange('timeField')}
        />
      </InlineField>

      <InlineField label="Message field" labelWidth={LABEL_WIDTH} interactive tooltip="Field shown as the log line body">
        <Input
          id="config-editor-message-field"
          width={INPUT_WIDTH}
          value={jsonData.messageField || ''}
          placeholder="_raw"
          onChange={onJsonChange('messageField')}
        />
      </InlineField>

      <InlineField label="Level field" labelWidth={LABEL_WIDTH} interactive tooltip="Field used for log level and volume grouping">
        <Input
          id="config-editor-level-field"
          width={INPUT_WIDTH}
          value={jsonData.levelField || ''}
          placeholder="level"
          onChange={onJsonChange('levelField')}
        />
      </InlineField>

      <InlineField label="Max lines" labelWidth={LABEL_WIDTH} interactive tooltip="Default maximum number of log lines per query">
        <Input
          id="config-editor-max-lines"
          width={INPUT_WIDTH}
          type="number"
          value={jsonData.maxLines ?? ''}
          placeholder="1000"
          onChange={onMaxLinesChange}
        />
      </InlineField>
    </Stack>
  );
}
