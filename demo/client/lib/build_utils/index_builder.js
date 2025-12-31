import fs from "fs";

class IndexBuilder {
    constructor() { 
        this.html = '';
        this.Js = '';
        this.Css = '';
    }
    readHTML(path) {
        this.html =  fs.readFileSync(path, 'utf8');
        return this;
    }
    embedJS(script) {
        // Strip trailing newline if present
        const trimmedScript = script.endsWith('\n') ? script.slice(0, -1) : script;
        this.Js += trimmedScript;
        return this;
    }
    embedJsFromFile(path) {
        const script = fs.readFileSync(path, 'utf8');
        return this.embedJS(script);
    }
    embedCSS(style) {
        this.Css += style;
        return this;
    }
    writeToFile(path) {
        this.html = this.html.replace(
            '<embedded-css></embedded-css>',
            `<style>${this.Css}</style>`
        );
        this.html = this.html.replace(
            '<embedded-js></embedded-js>',
            `<script>${this.Js}</script>`
        );


        fs.writeFileSync(path, this.html);
        this.html = null;
        this.Js = '';
        this.Css = '';
        return this;
    }
}

export { IndexBuilder };