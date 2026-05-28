import React, { ChangeEvent, useRef } from 'react';
import { CodeEditor, InlineField, Input, Monaco, monacoTypes, RadioButtonGroup, Stack } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from '../datasource';
import { LynxDataSourceOptions, LynxQuery, LynxQueryType } from '../types';
import { LANG_ID, registerLynxLanguage, validateQuery } from '../lynxLanguage';

type Props = QueryEditorProps<DataSource, LynxQuery, LynxDataSourceOptions>;

const QUERY_TYPES: Array<SelectableValue<LynxQueryType>> = [
  { label: 'Logs', value: 'logs' },
  { label: 'Metrics', value: 'metrics' },
];

export function QueryEditor({ query, onChange, onRunQuery, datasource }: Props) {
  const editorRef = useRef<monacoTypes.editor.IStandaloneCodeEditor>();
  const monacoRef = useRef<Monaco>();

  const onTypeChange = (value: LynxQueryType) => {
    onChange({ ...query, queryType: value });
    onRunQuery();
  };

  const onMaxLinesChange = (event: ChangeEvent<HTMLInputElement>) => {
    const raw = event.currentTarget.value;
    onChange({ ...query, maxLines: raw === '' ? undefined : Number(raw) });
  };

  const handleMount = (editor: monacoTypes.editor.IStandaloneCodeEditor, monaco: Monaco) => {
    editorRef.current = editor;
    monacoRef.current = monaco;
    registerLynxLanguage(monaco, datasource);
  };

  const handleBlur = (value: string) => {
    onChange({ ...query, queryText: value });
    if (monacoRef.current && editorRef.current) {
      validateQuery(monacoRef.current, editorRef.current, datasource, value);
    }
    onRunQuery();
  };

  return (
    <Stack direction="column" gap={1}>
      <Stack gap={1} alignItems="center">
        <InlineField label="Query type">
          <RadioButtonGroup options={QUERY_TYPES} value={query.queryType ?? 'logs'} onChange={onTypeChange} />
        </InlineField>
        {query.queryType !== 'metrics' && (
          <InlineField label="Max lines" tooltip="Maximum log lines to return">
            <Input
              type="number"
              width={12}
              value={query.maxLines ?? ''}
              placeholder="1000"
              onChange={onMaxLinesChange}
            />
          </InlineField>
        )}
      </Stack>
      <CodeEditor
        language={LANG_ID}
        value={query.queryText || ''}
        height={120}
        showMiniMap={false}
        showLineNumbers={false}
        monacoOptions={{ scrollBeyondLastLine: false, fontSize: 13, wordWrap: 'on' }}
        onEditorDidMount={handleMount}
        onBlur={handleBlur}
        onSave={handleBlur}
      />
    </Stack>
  );
}
