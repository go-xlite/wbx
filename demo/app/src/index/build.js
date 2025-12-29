import fs from "fs";


await Bun.build({
  entrypoints: ["./src/index/app.js"],
  outdir: "./dist/index",
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


fs.copyFileSync("./src/index/index.html", "./dist/index/index.html");