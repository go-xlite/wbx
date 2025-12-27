console.log('Index app.js loaded successfully!');
console.log('Served via virtual directory pattern: /index/p/app.js -> dist/index/app.js');

document.addEventListener('DOMContentLoaded', function() {
    const h1 = document.querySelector('h1');
    if (h1) {
        h1.style.color = '#2196F3';
    }
});