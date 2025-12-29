import fs from "fs";



await Bun.build({
  entrypoints: ["./src/stream-test/app.js"],
  outdir: "./dist/stream-test",
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

fs.copyFileSync("./src/stream-test/index.html", "./dist/stream-test/index.html");
fs.copyFileSync("./src/stream-test/styles.css", "./dist/stream-test/styles.css");