// demo file
export default async function defaultFunc(arg: any, signal: AbortSignal) {
  console.info('we have', { arg, signal });

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
