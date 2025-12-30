await Bun.build({
  entrypoints: ["./handler_ws/app/ws-manager-impl.js"],
  outdir: "./handler_ws/app-dist",
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


await Bun.build({
  entrypoints: ["./handler_ws/app/ws-shared-worker.js"],
  outdir: "./handler_ws/app-dist",
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
