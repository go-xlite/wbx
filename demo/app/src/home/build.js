import fs from "fs";


await Bun.build({
  entrypoints: ["./src/home/app.js"],
  outdir: "./dist/home",
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


fs.copyFileSync("./src/home/index.html", "./dist/home/index.html");