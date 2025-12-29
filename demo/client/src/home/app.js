console.log('Home app.js loaded successfully!');
console.log('Served via virtual directory pattern: /home/p/app.js -> dist/home/app.js');

document.addEventListener('DOMContentLoaded', function() {
    const h1 = document.querySelector('h1');
    if (h1) {
        h1.style.color = '#4CAF50';
    }
});