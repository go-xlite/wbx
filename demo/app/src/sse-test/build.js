import fs from "fs";


await Bun.build({
  entrypoints: ["../../webcast/js/sse-manager-impl.js"],
  outdir: "./dist/sse-test",
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


await Bun.build({
  entrypoints: ["./src/sse-test/app.js"],
  outdir: "./dist/sse-test",
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
  external: ["./sse-manager-impl.js"]
});

fs.copyFileSync("./src/sse-test/index.html", "./dist/sse-test/index.html");
fs.copyFileSync("./src/sse-test/styles.css", "./dist/sse-test/styles.css");