import { computed, h, inject, provide, ref } from 'vue';
import { detectOrganizationSlug } from '../api';

const RouterSymbol = Symbol('simple-router');

function readLocation() {
  if (typeof window === 'undefined') {
    return { path: '/', search: '', hash: '', organizationSlug: '', rawPath: '/' };
  }

  const rawPath = window.location.pathname || '/';
  const search = window.location.search || '';
  const hash = window.location.hash || '';
  const organizationSlug = detectOrganizationSlug(rawPath, search);
  const normalizedPath = (() => {
    if (organizationSlug && rawPath.startsWith(`/${organizationSlug}`)) {
      const trimmed = rawPath.slice(organizationSlug.length + 1);
      return trimmed ? `/${trimmed.replace(/^\/+/, '')}` : '/';
    }
    return rawPath || '/';
  })();

  return {
    path: normalizedPath,
    search,
    hash,
    organizationSlug,
    rawPath,
  };
}

function normalizeTarget(target) {
  if (typeof target !== 'string') {
    return '/';
  }

  try {
    const resolved = new URL(target, typeof window !== 'undefined' ? window.location.origin : 'http://localhost');
    return `${resolved.pathname}${resolved.search}${resolved.hash}`;
  } catch (error) {
    console.warn('Percorso non valido, fallback a "/"', error);
    return '/';
  }
}

function applySlugToTarget(normalizedTarget, organizationSlug) {
  if (!organizationSlug || /^https?:\/\//i.test(normalizedTarget)) {
    return normalizedTarget;
  }

  const [pathAndSearch, hash = ''] = normalizedTarget.split('#', 2);
  const [pathPart, searchPart = ''] = pathAndSearch.split('?', 2);
  const safePath = pathPart.startsWith('/') ? pathPart : `/${pathPart}`;
  const firstSegment = safePath
    .split('/')
    .map((segment) => segment.trim())
    .filter(Boolean)[0];

  if (firstSegment === organizationSlug || firstSegment === 'master') {
    return normalizedTarget;
  }

  const prefixedPath = `/${organizationSlug}${safePath}`;
  const searchSuffix = searchPart ? `?${searchPart}` : '';
  const hashSuffix = hash ? `#${hash}` : '';
  return `${prefixedPath}${searchSuffix}${hashSuffix}`;
}

function findMatchingRoute(path, routes) {
  const exact = routes.find((route) => route.path === path);
  if (exact) {
    return exact;
  }

  const wildcard = routes.find((route) => route.path.endsWith('*') && path.startsWith(route.path.replace('*', '')));
  if (wildcard) {
    return wildcard;
  }

  return routes.find((route) => route.path === '*');
}

function resolveRoute(path, routes, location) {
  const matched = findMatchingRoute(path, routes) ?? routes[0];
  if (matched?.redirect) {
    const redirectTarget = typeof matched.redirect === 'function' ? matched.redirect(location) : matched.redirect;
    return resolveRoute(redirectTarget, routes, location);
  }

  return {
    ...matched,
    location,
    slug: location.organizationSlug || '',
    fullPath: `${location.path}${location.search}${location.hash}`,
  };
}

export function createRouter(options) {
  const location = ref(readLocation());

  const currentRoute = computed(() => resolveRoute(location.value.path, options.routes, location.value));

  function navigate(target, replace = false) {
    const normalized = normalizeTarget(target);
    const withSlug = applySlugToTarget(normalized, location.value.organizationSlug);
    if (typeof window === 'undefined') {
      location.value = readLocation();
      return;
    }

    if (replace) {
      window.history.replaceState({}, '', withSlug);
    } else {
      window.history.pushState({}, '', withSlug);
    }
    location.value = readLocation();
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('popstate', () => {
      location.value = readLocation();
    });
  }

  const router = {
    currentRoute,
    routes: options.routes,
    push: (target) => navigate(target, false),
    replace: (target) => navigate(target, true),
  };

  router.install = (app) => {
    app.provide(RouterSymbol, router);
    app.component('RouterView', RouterView);
    app.component('RouterLink', RouterLink);
  };

  return router;
}

export function useRouter() {
  const router = inject(RouterSymbol);
  if (!router) {
    throw new Error('Router non disponibile: assicurati di chiamare app.use(router).');
  }
  return router;
}

export function useRoute() {
  const router = useRouter();
  return router.currentRoute;
}

const RouterView = {
  name: 'RouterView',
  setup() {
    const route = useRoute();
    return () => {
      const component = route.value?.component;
      const routeProps =
        typeof route.value?.props === 'function' ? route.value.props(route.value) : route.value?.props;
      return component ? h(component, { route: route.value, ...(routeProps || {}) }) : null;
    };
  },
};

const RouterLink = {
  name: 'RouterLink',
  props: {
    to: {
      type: String,
      required: true,
    },
    customClass: {
      type: String,
      default: '',
    },
    activeClass: {
      type: String,
      default: 'is-active',
    },
    exact: {
      type: Boolean,
      default: false,
    },
  },
  setup(props, { slots }) {
    const router = useRouter();
    const route = useRoute();

    const isActive = computed(() => {
      const targetPath = normalizeTarget(props.to);
      if (props.exact) {
        return route.value?.fullPath === targetPath;
      }
      return route.value?.fullPath.startsWith(targetPath);
    });

    const navigate = (event) => {
      event?.preventDefault();
      router.push(props.to);
    };

    return () =>
      h(
        'a',
        {
          href: props.to,
          class: [props.customClass, isActive.value ? props.activeClass : ''].join(' ').trim(),
          onClick: navigate,
        },
        slots.default ? slots.default() : props.to,
      );
  },
};
