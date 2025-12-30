
(() => {
document.addEventListener('DOMContentLoaded', function () {
    // Use getAttribute to get raw href value before browser resolves it
    document.querySelectorAll('a[href*="__PREFIX__"]').forEach(el => {
        const rawHref = el.getAttribute('href');
        const newHref = rawHref.replace(/__PREFIX__/g, window.PATH_PREFIX);
        el.setAttribute('href', newHref);
    });
}, { once: true });
 const segments = window.location.pathname.split('/').filter(s => s);
    window.PATH_PREFIX = segments.length > 0 ? '/' + segments[0] : '';
})()

