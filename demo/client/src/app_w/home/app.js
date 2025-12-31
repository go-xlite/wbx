import { loadServiceWorker } from '../../../lib/site_core.js';

//  loadServiceWorker();

document.addEventListener('DOMContentLoaded', function() {
    const h1 = document.querySelector('h1');
    if (h1) {
        h1.style.color = '#4CAF50';
    }
});