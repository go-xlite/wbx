
await Bun.build({
  entrypoints: ["./sse-manager-impl-raw.js"],
  outdir: "./dist",
  minify: {
    whitespace: true,
    identifiers: true,
    syntax: true,
    properties: true,
    keepNames: false
  },
  sourcemap: "none",
  target: "browser",
  splitting: false,
  naming: "[dir]/[name].min.[ext]"
});