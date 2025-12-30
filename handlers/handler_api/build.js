await Bun.build({
  entrypoints: ["./handler_api/app/api-core.js"],
  outdir: "./handler_api/app-dist",
  minify: {
    whitespace: true,
    identifiers: true,
    syntax: true,
    properties: true,
    keepNames: false
  },
//   drop: ["console"],
  sourcemap: "none",
  target: "browser",
  splitting: false
});