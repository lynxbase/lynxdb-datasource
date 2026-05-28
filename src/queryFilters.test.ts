import { applyAdHocFilters, appendWhere, escapeValue, filterClause } from './queryFilters';

describe('queryFilters', () => {
  it('escapes quotes and backslashes', () => {
    expect(escapeValue('a"b\\c')).toBe('a\\"b\\\\c');
  });

  it('builds equality and inequality clauses', () => {
    expect(filterClause('level', '=', 'ERROR')).toBe('level="ERROR"');
    expect(filterClause('level', '!=', 'INFO')).toBe('level!="INFO"');
    expect(filterClause('', '=', 'x')).toBeUndefined();
  });

  it('appends where to a non-empty query and uses a bare search otherwise', () => {
    expect(appendWhere('_source=nginx', 'status>=500')).toBe('_source=nginx | where status>=500');
    expect(appendWhere('   ', 'level="ERROR"')).toBe('level="ERROR"');
  });

  it('joins ad-hoc filters with and', () => {
    const filters = [
      { key: 'level', operator: '=', value: 'ERROR' },
      { key: 'host', operator: '!=', value: 'web-01' },
    ];
    expect(applyAdHocFilters('*', filters)).toBe('* | where level="ERROR" and host!="web-01"');
    expect(applyAdHocFilters('*', [])).toBe('*');
    expect(applyAdHocFilters('*')).toBe('*');
  });
});
