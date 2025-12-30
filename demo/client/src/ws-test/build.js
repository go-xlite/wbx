import fs from "fs";
import { transform } from 'lightningcss';
import { IndexBuilder } from "../../lib/builder.js";


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

const css = fs.readFileSync("./src/ws-test/styles.css", 'utf8');
const { code } = transform({
  filename: 'style.css',
  code: Buffer.from(css),
  minify: true,
});

new IndexBuilder()
  .readHTML("./src/ws-test/index.html")
  .embedCSS(code.toString())
  .embedJsFromFile("./lib-dist/embed.js")
  .writeToFile("./dist/ws-test/index.html");
