return async (ctx) => {
  const log = ctx.logger('read-demo2');
  const content = ctx.fs.readFile('tmp/merge_demo/demo2.txt');
  log.info('[READ_RESULT]' + JSON.stringify(content));
  return { content };
};