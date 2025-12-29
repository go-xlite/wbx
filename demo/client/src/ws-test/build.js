import fs from "fs";
import { transform } from 'lightningcss';


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

fs.writeFileSync("./dist/ws-test/styles.css", code);
fs.copyFileSync("./src/ws-test/index.html", "./dist/ws-test/index.html");