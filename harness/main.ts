import { buildWriter, FdReader } from './fd.ts';

export type Request = {
  import: string;
  method?: string; // or 'default'
  id: string | number;
  args?: any[];
};

const r = new FdReader(3);
const w = buildWriter(4);
const write = (payload: any) => w(JSON.stringify(payload), '\n');

const active = new Set<number | string>();

for (;;) {
  const line = await r.line();
  const raw = JSON.parse(line.toString('utf-8'));

  const payload = raw as Request;
  if (active.has(payload.id)) {
    throw new Error('req already active: ' + payload.id);
  }
  active.add(payload.id);

  (async () => {
    try {
      const module = await import(payload.import);
      const method = module[payload.method ?? 'default'];
      const res = await method(...(payload.args ?? []));
      await write({ status: 'ok', id: payload.id, res });
    } catch (e) {
      console.warn(e);
      // TODO: stringify error
      await write({ status: 'err', id: payload.id });
    } finally {
      active.delete(payload.id);
    }
  })();
}
