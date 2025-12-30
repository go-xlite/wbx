import fs from "fs";
import { transform } from 'lightningcss';
import { IndexBuilder } from "../../lib/builder.js";


await Bun.build({
  entrypoints: ["./src/api-demo/app.js"],
  outdir: "./dist/api-demo",
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


const css = fs.readFileSync("./src/api-demo/styles.css", 'utf8');
const { code } = transform({
  filename: 'style.css',
  code: Buffer.from(css),
  minify: true,
});

new IndexBuilder()
  .readHTML("./src/api-demo/index.html")
  .embedCSS(code.toString())
  .embedJsFromFile("./lib-dist/embed.js")
  .writeToFile("./dist/api-demo/index.html");