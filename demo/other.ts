const abortSignalSymbol = Symbol.for('nodejs-holder.signal');

// demo file
export default async function defaultFunc() {
  const signal = defaultFunc[abortSignalSymbol] as AbortSignal;
  console.info('we have signal', signal);

  if (signal) {
    signal.addEventListener('abort', () => {
      console.info('got abort from caller');
    });

    await new Promise((r) => {
      // TODO: never resolves
    });
  }

  return 'lol hi ' + Math.random();
}
