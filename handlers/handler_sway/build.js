await Bun.build({
  entrypoints: ["./handler_xapp/app/sway-core.js"],
  outdir: "./handler_xapp/app-dist",
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