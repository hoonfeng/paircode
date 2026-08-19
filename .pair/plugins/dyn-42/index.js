return async (ctx) => {
  const content = ctx.fs.readFile('tmp/merge_demo/demo2.txt');
  console.log('[READ_RESULT]' + JSON.stringify(content));
  return { content };
};