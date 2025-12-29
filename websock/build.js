import fs from "fs";

await Bun.build({
  entrypoints: ["./js/browser-ws-manager.js"],
  outdir: "./client",
  minify: {
    whitespace: true,
    identifiers: true,
    syntax: true,
    properties: true,
    keepNames: false,
  },
  drop: ["console"],
  sourcemap: "none",
  target: "browser",
  splitting: false
});

await Bun.build({
  entrypoints: ["./js/browser-shared-worker.js"],
  outdir: "./client",
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
