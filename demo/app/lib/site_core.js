

const pathParts = window.location.pathname.split('/').filter(p => p);
const pathPrefix = pathParts.length > 0 ? '/' + pathParts[0] : '';


export { pathPrefix }