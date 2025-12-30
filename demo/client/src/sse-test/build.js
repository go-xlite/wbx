import fs from "fs";
import { transform } from 'lightningcss';
import { IndexBuilder } from "../../lib/builder.js";


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
  splitting: false
});


const css = fs.readFileSync("./src/sse-test/styles.css", 'utf8');
const { code } = transform({
  filename: 'style.css',
  code: Buffer.from(css),
  minify: true,
});

new IndexBuilder()
  .readHTML("./src/sse-test/index.html")
  .embedCSS(code.toString())
  .embedJsFromFile("./lib-dist/embed.js")
  .writeToFile("./dist/sse-test/index.html");

