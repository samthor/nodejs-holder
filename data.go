package main

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
var r = new FdReader(3);
var w = buildWriter(4);
var write2 = (payload) => w(JSON.stringify(payload), "\n");
var active = /* @__PURE__ */ new Set();
for (; ; ) {
  const line = await r.line();
  const raw = JSON.parse(line.toString("utf-8"));
  const payload = raw;
  if (active.has(payload.id)) {
    throw new Error("req already active: " + payload.id);
  }
  active.add(payload.id);
  (async () => {
    try {
      const module = await import(payload.import);
      const method = module[payload.method ?? "default"];
      const res = await method(...payload.args ?? []);
      await write2({ status: "ok", res });
    } catch (e) {
      console.warn(e);
      await write2({ status: "err" });
    } finally {
      active.delete(payload.id);
    }
  })();
}
	`
)
