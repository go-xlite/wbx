import fs from "fs";


await Bun.build({
  entrypoints: ["./src/ws-test/app.js"],
  outdir: "./dist/ws-test",
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

fs.copyFileSync("./src/ws-test/index.html", "./dist/ws-test/index.html");
fs.copyFileSync("./src/ws-test/styles.css", "./dist/ws-test/styles.css");