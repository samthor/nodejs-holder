package lib

const (
  jsHarness = `
// harness/fd.ts
import * as fs from "node:fs";
var MIN_BUFFER = 1024;
var MAX_BUFFER = 1 << 16;
var NEWLINE = 10;
var FdReader = class {
  constructor(fd) {
    this.fd = fd;
    this.at = 0;
    this.buf = Buffer.alloc(MIN_BUFFER);
  }
  async line() {
    for (; ; ) {
      const rem = this.buf.length - this.at;
      if (rem < MIN_BUFFER || rem < this.buf.length / 2) {
        if (this.buf.length + 1 >= MAX_BUFFER) {
          throw new Error("buffer too big: " + this.buf.length);
        }
        const extra = Buffer.alloc(Math.max(MIN_BUFFER, this.buf.length * 2));
        extra.set(this.buf);
        this.buf = extra;
        continue;
      }
      const p = new Promise((resolve, reject) => {
        fs.read(this.fd, this.buf, this.at, rem, -1, (err, bytesRead2) => {
          err ? reject(err) : resolve(bytesRead2);
        });
      });
      const bytesRead = await p;
      if (bytesRead <= 0) {
        throw new Error("bad bytesRead: " + bytesRead);
      }
      const newlineAt = this.buf.indexOf(NEWLINE, this.at);
      this.at += bytesRead;
      if (newlineAt === -1) {
        continue;
      }
      const out = this.buf.subarray(0, newlineAt);
      this.buf = this.buf.subarray(newlineAt + 1);
      this.at = 0;
      return out;
    }
  }
};
async function writeAll(fd, payload) {
  while (payload.length) {
    const p = new Promise((resolve, reject) => {
      fs.write(fd, payload, (err, bytesWritten2) => {
        err ? reject(err) : resolve(bytesWritten2);
      });
    });
    const bytesWritten = await p;
    if (bytesWritten <= 0) {
      throw new Error("bad bytesWritten: " + bytesWritten);
    }
    payload = payload.subarray(bytesWritten);
  }
}
function buildWriter(fd) {
  let p = Promise.resolve();
  return async (...payload) => {
    p = p.then(async () => {
      for (let part of payload) {
        if (typeof part === "string") {
          part = Buffer.from(part, "utf-8");
        }
        await writeAll(fd, part);
      }
    });
    return p;
  };
}

// harness/main.ts
process.title = "nodejs-holder";
var abortSignalSymbol = Symbol.for("nodejs-holder.signal");
var r = new FdReader(3);
var w = buildWriter(4);
var write2 = (payload) => w(JSON.stringify(payload), "\n");
var active = /* @__PURE__ */ new Map();
for (; ; ) {
  const bytes = await r.line();
  const line = bytes.toString("utf-8");
  if (!line) {
    continue;
  }
  const raw = JSON.parse(line);
  const payload = raw;
  if (payload.cancel) {
    if (!payload.id) {
      process.exit(0);
    }
    const h = active.get(payload.id);
    h?.();
    continue;
  }
  if (active.has(payload.id)) {
    throw new Error("req already active: " + payload.id);
  }
  const c = new AbortController();
  active.set(payload.id, () => c.abort());
  runRequest(payload, c.signal).finally(() => active.delete(payload.id));
}
async function runRequest(payload, signal) {
  try {
    const module = await import(payload.import);
    const method = module[payload.method ?? "default"];
    method[abortSignalSymbol] = signal;
    Promise.resolve().then(() => {
      if (method[abortSignalSymbol] === signal) {
        delete method[abortSignalSymbol];
      }
    });
    const res = await method(...payload.args ?? []);
    const r2 = {
      id: payload.id,
      status: "ok",
      res
    };
    await write2(r2);
  } catch (e) {
    const r2 = {
      id: payload.id,
      status: "err",
      errtext: String(e)
    };
    await write2(r2);
  }
}
  `
)

