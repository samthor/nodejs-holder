import * as fs from 'node:fs';

const MIN_BUFFER = 1024;
const MAX_BUFFER = 1 << 16; // 65k
const NEWLINE = 10;

export class FdReader {
  private at = 0;
  private buf: Buffer = Buffer.alloc(MIN_BUFFER);

  constructor(public readonly fd: number) {}

  async line(): Promise<Buffer> {
    for (;;) {
      const rem = this.buf.length - this.at;
      if (rem < MIN_BUFFER || rem < this.buf.length / 2) {
        if (this.buf.length + 1 >= MAX_BUFFER) {
          throw new Error('buffer too big: ' + this.buf.length);
        }

        // double and clone
        const extra = Buffer.alloc(Math.max(MIN_BUFFER, this.buf.length * 2));
        extra.set(this.buf);
        this.buf = extra;
        continue;
      }

      const p = new Promise<number>((resolve, reject) => {
        // TODO: EOF?
        fs.read(this.fd, this.buf, this.at, rem, -1, (err, bytesRead) => {
          err ? reject(err) : resolve(bytesRead);
        });
      });
      const bytesRead = await p;
      if (bytesRead <= 0) {
        throw new Error('bad bytesRead: ' + bytesRead);
      }

      // look for newline in new data
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
}

export async function writeAll(fd: number, payload: Buffer) {
  while (payload.length) {
    const p = new Promise<number>((resolve, reject) => {
      fs.write(fd, payload, (err, bytesWritten) => {
        err ? reject(err) : resolve(bytesWritten);
      });
    });

    const bytesWritten = await p;
    if (bytesWritten <= 0) {
      throw new Error('bad bytesWritten: ' + bytesWritten);
    }

    payload = payload.subarray(bytesWritten);
  }
}

export function buildWriter(fd: number) {
  let p = Promise.resolve();

  return async (...payload: (Buffer | string)[]) => {
    p = p.then(async () => {
      for (let part of payload) {
        if (typeof part === 'string') {
          part = Buffer.from(part, 'utf-8');
        }
        await writeAll(fd, part);
      }
    });

    return p;
  };
}
