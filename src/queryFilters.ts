import { AdHocVariableFilter } from '@grafana/data';

// escapeValue makes a value safe to place inside a double-quoted Lynx Flow string.
export function escapeValue(value: string): string {
  return String(value).replace(/\\/g, '\\\\').replace(/"/g, '\\"');
}

// filterClause builds a single Lynx Flow predicate, e.g. level="ERROR".
export function filterClause(key: string, operator: string, value: string): string | undefined {
  if (!key) {
    return undefined;
  }
  const v = escapeValue(value);
  switch (operator) {
    case '!=':
      return `${key}!="${v}"`;
    case '=':
    case '==':
    default:
      return `${key}="${v}"`;
  }
}

// appendWhere adds a where clause to an existing query, or returns the clause as
// a bare search when the query is empty.
export function appendWhere(queryText: string | undefined, clause: string): string {
  const base = (queryText ?? '').trim();
  return base ? `${base} | where ${clause}` : clause;
}

// applyAdHocFilters appends dashboard ad-hoc filters to a query, joined with and.
export function applyAdHocFilters(queryText: string | undefined, filters?: AdHocVariableFilter[]): string | undefined {
  if (!filters || filters.length === 0) {
    return queryText;
  }
  const clauses = filters
    .map((f) => filterClause(f.key, f.operator, f.value))
    .filter((c): c is string => Boolean(c));
  if (clauses.length === 0) {
    return queryText;
  }
  return appendWhere(queryText, clauses.join(' and '));
}
