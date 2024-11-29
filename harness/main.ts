import { buildWriter, FdReader } from './fd.ts';

export type Request = {
  id: string;
  import: string;
  method?: string; // or 'default'
  cancel?: true;
  arg?: any;
};

export type Response = {
  id: string;
  status: string;
  res?: any;
  errtext?: string;
};

process.title = 'nodejs-holder';

const r = new FdReader(3);
const w = buildWriter(4);
const write = (payload: any) => w(JSON.stringify(payload), '\n');

const active = new Map<string, () => void>();

for (;;) {
  const bytes = await r.line();
  const line = bytes.toString('utf-8');
  if (!line) {
    continue; // should never happen but check anyway
  }
  const raw = JSON.parse(line);

  const payload = raw as Request;
  if (payload.cancel) {
    if (!payload.id) {
      // aggressive shutdown
      process.exit(0);
    }

    const h = active.get(payload.id);
    h?.(); // ignore if missing, maybe racey
    continue;
  }

  if (active.has(payload.id)) {
    throw new Error('req already active: ' + payload.id);
  }

  const c = new AbortController();
  active.set(payload.id, () => c.abort());

  // async error ignored
  runRequest(payload, c.signal).finally(() => active.delete(payload.id));
}

async function runRequest(payload: Request, signal: AbortSignal) {
  try {
    const module = await import(payload.import);
    const method = module[payload.method ?? 'default'];

    const res = await method(payload.arg, signal);
    const r: Response = {
      id: payload.id,
      status: 'ok',
      res,
    };
    await write(r);
  } catch (e) {
    // stringify err and send response
    const r: Response = {
      id: payload.id,
      status: 'err',
      errtext: String(e),
    };
    await write(r);
  }
}
