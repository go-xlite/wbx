import fs from "fs";
import { transform } from 'lightningcss';
import { IndexBuilder } from "../../lib/builder.js";

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

const css = fs.readFileSync("./src/stream-test/styles.css", 'utf8');
const { code } = transform({
  filename: 'style.css',
  code: Buffer.from(css),
  minify: true,
});

new IndexBuilder()
  .readHTML("./src/stream-test/index.html")
  .embedCSS(code.toString())
  .embedJsFromFile("./lib-dist/embed.js")
  .writeToFile("./dist/stream-test/index.html");
