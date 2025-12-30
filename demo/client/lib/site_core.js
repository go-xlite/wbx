

const pathParts = window.location.pathname.split('/').filter(p => p);
const pathPrefix = pathParts.length > 0 ? '/' + pathParts[0] : '';

function loadServiceWorker() {
    // Register Service Worker
    if ('serviceWorker' in navigator) {
        console.log('[App] ServiceWorker supported');

        const registerServiceWorker = () => {
            const swPath = `${window.PATH_PREFIX || ''}/index/p/sw.js`;

            navigator.serviceWorker.register(swPath, { scope: `${window.PATH_PREFIX || ''}/` })
                .then(registration => {
                    // Check for updates
                    registration.addEventListener('updatefound', () => {
                        const newWorker = registration.installing;
                        newWorker.addEventListener('statechange', () => {
                            if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                                console.log('[App] New ServiceWorker available, please refresh');
                            }
                        });
                    });
                })
                .catch(error => {
                    console.error('[App] ServiceWorker registration failed:', error);
                });
        };

        // Register immediately if page already loaded, otherwise wait for load event
        if (document.readyState === 'complete') {
            registerServiceWorker();
        } else {
            window.addEventListener('load', registerServiceWorker);
        }
    }
}


export { pathPrefix, loadServiceWorker }