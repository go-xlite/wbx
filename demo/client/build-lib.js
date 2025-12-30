
await Bun.build({
  entrypoints: ["./lib/embed.js"],
  outdir: "./lib-dist",
  minify: {
    whitespace: true,
    identifiers: true,
    syntax: true,
    properties: true,
    keepNames: false
  },
  sourcemap: "none",
  target: "browser",
  splitting: false
});
