import { Monaco, monacoTypes } from '@grafana/ui';
import type { DataSource } from './datasource';

export const LANG_ID = 'lynxflow';

// Lynx Flow / SPL2 pipeline commands and clause keywords (from `lynxdb grammar ebnf`).
const COMMANDS = [
  'from',
  'index',
  'search',
  'where',
  'stats',
  'eval',
  'sort',
  'head',
  'tail',
  'reverse',
  'table',
  'fields',
  'rename',
  'dedup',
  'rex',
  'regex',
  'replace',
  'bin',
  'timechart',
  'top',
  'rare',
  'fillnull',
  'streamstats',
  'eventstats',
  'join',
  'append',
  'union',
  'transaction',
  'mvexpand',
  'makeresults',
  'parse',
  'lookup',
  'percentiles',
  'rank',
  'bottom',
  'by',
  'as',
  'with',
  'in',
  'span',
  'and',
  'or',
  'not',
];

// Aggregation functions usable in stats/timechart (from `lynxdb grammar ebnf`).
const FUNCTIONS = [
  'count',
  'sum',
  'sumsq',
  'avg',
  'mean',
  'min',
  'max',
  'dc',
  'distinct_count',
  'estdc',
  'values',
  'list',
  'mode',
  'first',
  'last',
  'stdev',
  'stdevp',
  'var',
  'varp',
  'range',
  'median',
  'perc',
  'percentile',
  'earliest',
  'latest',
  'rate',
  'per_second',
  'per_minute',
  'per_hour',
  'per_day',
];

const monarch: monacoTypes.languages.IMonarchLanguage = {
  defaultToken: '',
  ignoreCase: true,
  keywords: COMMANDS,
  functions: FUNCTIONS,
  tokenizer: {
    root: [
      [/\|/, 'operator'],
      [
        /[a-zA-Z_][\w.]*/,
        { cases: { '@keywords': 'keyword', '@functions': 'predefined', '@default': 'identifier' } },
      ],
      [/"([^"\\]|\\.)*"/, 'string'],
      [/'([^'\\]|\\.)*'/, 'string'],
      [/\d+(\.\d+)?/, 'number'],
      [/[=!<>]=?|[+\-*/]/, 'operator'],
      [/[{}()[\]]/, '@brackets'],
      [/[ \t\r\n]+/, 'white'],
    ],
  },
};

let registered = false;
let activeDatasource: DataSource | undefined;

// registerLynxLanguage registers the Lynx Flow language, completion, and
// validation once per Monaco instance and tracks the datasource that the
// completion provider should query.
export function registerLynxLanguage(monaco: Monaco, datasource: DataSource) {
  activeDatasource = datasource;
  if (registered) {
    return;
  }
  registered = true;

  monaco.languages.register({ id: LANG_ID });
  monaco.languages.setMonarchTokensProvider(LANG_ID, monarch);
  monaco.languages.setLanguageConfiguration(LANG_ID, {
    autoClosingPairs: [
      { open: '"', close: '"' },
      { open: "'", close: "'" },
      { open: '(', close: ')' },
    ],
    brackets: [
      ['(', ')'],
      ['[', ']'],
      ['{', '}'],
    ],
  });

  monaco.languages.registerCompletionItemProvider(LANG_ID, {
    triggerCharacters: [' ', '|', '=', ':', '.'],
    async provideCompletionItems(model, position) {
      const word = model.getWordUntilPosition(position);
      const range: monacoTypes.IRange = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };

      const suggestions: monacoTypes.languages.CompletionItem[] = [];

      for (const kw of COMMANDS) {
        suggestions.push({ label: kw, kind: monaco.languages.CompletionItemKind.Keyword, insertText: kw, range });
      }
      for (const fn of FUNCTIONS) {
        suggestions.push({
          label: `${fn}()`,
          kind: monaco.languages.CompletionItemKind.Function,
          insertText: `${fn}($0)`,
          insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          range,
        });
      }

      const ds = activeDatasource;
      if (ds) {
        const textUntil = model.getValueInRange({
          startLineNumber: 1,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        });
        const fieldMatch = textUntil.match(/([A-Za-z_][\w.]*)\s*[:=]\s*"?([\w.\-*]*)$/);
        try {
          if (fieldMatch) {
            const values = await ds.getFieldValues(fieldMatch[1], 20);
            for (const v of values) {
              const text = String(v.value);
              suggestions.push({
                label: text,
                kind: monaco.languages.CompletionItemKind.Value,
                insertText: text,
                range,
              });
            }
          } else {
            const fields = await ds.getFields(word.word || undefined);
            for (const f of fields) {
              suggestions.push({
                label: f.name,
                kind: monaco.languages.CompletionItemKind.Field,
                insertText: f.name,
                detail: f.type,
                range,
              });
            }
          }
        } catch {
          // Autocomplete is best effort; ignore lookup failures.
        }
      }

      return { suggestions };
    },
  });
}

// validateQuery asks the backend to parse the query and renders parse errors as
// editor markers.
export async function validateQuery(
  monaco: Monaco,
  editor: monacoTypes.editor.IStandaloneCodeEditor,
  datasource: DataSource,
  text: string
) {
  const model = editor.getModel();
  if (!model) {
    return;
  }
  if (!text || !text.trim()) {
    monaco.editor.setModelMarkers(model, 'lynxdb', []);
    return;
  }

  try {
    const res = await datasource.explain(text);
    if (res?.is_valid !== false) {
      monaco.editor.setModelMarkers(model, 'lynxdb', []);
      return;
    }
    const markers = (res.errors ?? []).map((err) => {
      const start = model.getPositionAt(err.position ?? 0);
      const end = model.getPositionAt((err.position ?? 0) + (err.length ?? 1));
      return {
        severity: monaco.MarkerSeverity.Error,
        message: err.suggestion ? `${err.message} (${err.suggestion})` : err.message,
        startLineNumber: start.lineNumber,
        startColumn: start.column,
        endLineNumber: end.lineNumber,
        endColumn: end.column,
      };
    });
    monaco.editor.setModelMarkers(model, 'lynxdb', markers);
  } catch {
    // Validation is best effort; ignore failures.
  }
}
