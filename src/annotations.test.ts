import { createDataFrame, FieldType } from '@grafana/data';
import { framesToAnnotations } from './annotations';

describe('framesToAnnotations', () => {
  it('maps log frame rows to annotation events', () => {
    const frame = createDataFrame({
      fields: [
        { name: 'timestamp', type: FieldType.time, values: [1000, 2000] },
        { name: 'body', type: FieldType.string, values: ['boom', 'warn'] },
        { name: 'severity', type: FieldType.string, values: ['ERROR', ''] },
      ],
    });

    const events = framesToAnnotations([frame]);

    expect(events).toHaveLength(2);
    expect(events[0]).toMatchObject({ time: 1000, text: 'boom', tags: ['ERROR'] });
    expect(events[1].time).toBe(2000);
    expect(events[1].text).toBe('warn');
    expect(events[1].tags).toBeUndefined();
  });

  it('skips frames without a time field', () => {
    const frame = createDataFrame({
      fields: [{ name: 'host', type: FieldType.string, values: ['web-01'] }],
    });

    expect(framesToAnnotations([frame])).toHaveLength(0);
  });
});
