return async (ctx) => {
  const content = ctx.fs.readFile('tmp/merge_demo/demo2.txt');
  return { content };
};