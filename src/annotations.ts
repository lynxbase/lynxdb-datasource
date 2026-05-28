import { AnnotationEvent, AnnotationSupport, DataFrame, Field, FieldType } from '@grafana/data';
import { Observable, of } from 'rxjs';
import { LynxQuery } from './types';

// annotationSupport runs an annotation's query as a normal events query and maps
// the resulting log frames to annotation events (one per log line).
export const annotationSupport: AnnotationSupport<LynxQuery> = {
  getDefaultQuery: () => ({ queryText: '', queryType: 'logs' }),

  prepareQuery: (anno) => {
    // React annotations may store the query at the top level or under target.
    const q = ((anno as { target?: LynxQuery }).target ?? (anno as unknown as LynxQuery)) as LynxQuery;
    if (!q || !q.queryText) {
      return undefined;
    }
    return { ...q, refId: q.refId || 'Anno', queryType: 'logs' };
  },

  processEvents: (_anno, frames: DataFrame[]): Observable<AnnotationEvent[] | undefined> => {
    return of(framesToAnnotations(frames));
  },
};

export function framesToAnnotations(frames: DataFrame[]): AnnotationEvent[] {
  const events: AnnotationEvent[] = [];

  for (const frame of frames) {
    const timeField = frame.fields.find((f) => f.type === FieldType.time);
    if (!timeField) {
      continue;
    }
    const bodyField =
      frame.fields.find((f) => f.name === 'body') ?? frame.fields.find((f) => f.type === FieldType.string);
    const severityField = frame.fields.find((f) => f.name === 'severity');

    for (let i = 0; i < timeField.values.length; i++) {
      const rawTime = timeField.values[i];
      if (rawTime == null) {
        continue;
      }
      events.push({
        time: typeof rawTime === 'number' ? rawTime : new Date(rawTime).valueOf(),
        text: bodyField ? String(valueAt(bodyField, i)) : '',
        tags: tagsFor(severityField, i),
      });
    }
  }

  return events;
}

function valueAt(field: Field, i: number): unknown {
  const v = field.values[i];
  return v ?? '';
}

function tagsFor(field: Field | undefined, i: number): string[] | undefined {
  if (!field) {
    return undefined;
  }
  const v = field.values[i];
  return v ? [String(v)] : undefined;
}
