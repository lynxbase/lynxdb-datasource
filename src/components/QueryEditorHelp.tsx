import React from 'react';
import { QueryEditorHelpProps } from '@grafana/data';
import { LynxQuery, LynxQueryType } from '../types';

interface Example {
  title: string;
  queryText: string;
  queryType: LynxQueryType;
}

const EXAMPLES: Example[] = [
  { title: 'Errors from a source', queryText: 'level=error source=nginx', queryType: 'logs' },
  { title: 'Server errors (status >= 500)', queryText: 'status>=500', queryType: 'logs' },
  { title: 'Full-text search', queryText: 'search "connection refused"', queryType: 'logs' },
  { title: 'Count by host', queryText: '* | stats count() by host', queryType: 'metrics' },
  { title: 'Errors over time by level', queryText: '* | timechart count() span=1m by level', queryType: 'metrics' },
  { title: 'Top URIs', queryText: 'source=nginx | top 10 uri', queryType: 'metrics' },
];

export function QueryEditorHelp({ onClickExample }: QueryEditorHelpProps<LynxQuery>) {
  return (
    <div>
      <h4>Lynx Flow / SPL2 examples</h4>
      <p>Click an example to use it. Filters can be combined; pipe into stats or timechart for metrics.</p>
      {EXAMPLES.map((ex) => (
        <div key={ex.queryText} style={{ marginBottom: 8 }}>
          <button
            type="button"
            className="gf-form-label query-keyword pointer"
            onClick={() =>
              onClickExample({
                refId: 'A',
                queryText: ex.queryText,
                queryType: ex.queryType,
              })
            }
          >
            {ex.title}
          </button>{' '}
          <code>{ex.queryText}</code>
        </div>
      ))}
    </div>
  );
}
