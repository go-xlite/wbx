await Bun.build({
  entrypoints: ["./handler_sse/app/sse-manager-impl.js"],
  outdir: "./handler_sse/app-dist",
  minify: {
    whitespace: true,
    identifiers: true,
    syntax: true,
    properties: true,
    keepNames: false
  },
  drop: ["console"],
  sourcemap: "none",
  target: "browser",
  splitting: false
});