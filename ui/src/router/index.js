import { createRouter, createWebHistory } from 'vue-router'

const EmptyRouteView = {
  render: () => null,
}

const appTitle = 'ComicHero'
const routeAccess = {
  loaded: false,
  canAccessMetron: false,
  isAdmin: false,
  hasUser: false,
  readOnlyGuest: false,
}

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: { name: 'dashboard' } },
    { path: '/readingOrders', redirect: { name: 'readingOrders' } },
    { path: '/readingOrders/new', redirect: { name: 'readingOrdersNew' } },
    {
      path: '/verify-email',
      name: 'verifyEmail',
      component: EmptyRouteView,
      meta: { eyebrow: 'Account', title: 'Verify email' },
    },
    {
      path: '/reset-password',
      name: 'resetPassword',
      component: EmptyRouteView,
      meta: { eyebrow: 'Account', title: 'Reset password' },
    },
    {
      path: '/readingOrders/:id',
      redirect: (to) => ({ name: 'readingOrderDetail', params: { id: to.params.id } }),
    },
    {
      path: '/readingOrders/:id/edit',
      redirect: (to) => ({ name: 'readingOrderEdit', params: { id: to.params.id } }),
    },
    { path: '/achievements', redirect: { name: 'progress' } },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: EmptyRouteView,
      meta: { eyebrow: 'Dashboard', title: 'Dashboard', requiresUser: true },
    },
    {
      path: '/reading-orders',
      name: 'readingOrders',
      component: EmptyRouteView,
      meta: { eyebrow: 'Reading Orders', title: 'Reading Orders', showCount: true },
    },
    {
      path: '/reading-orders/new',
      name: 'readingOrdersNew',
      component: EmptyRouteView,
      meta: { eyebrow: 'Reading Orders', title: 'New reading order' },
    },
    {
      path: '/reading-orders/:id',
      name: 'readingOrderDetail',
      component: EmptyRouteView,
      meta: { eyebrow: 'Reading Orders', title: 'Reading order' },
    },
    {
      path: '/reading-orders/:id/edit',
      name: 'readingOrderEdit',
      component: EmptyRouteView,
      meta: { eyebrow: 'Reading Orders', title: 'Edit reading order' },
    },
    {
      path: '/arcs',
      name: 'arcs',
      component: EmptyRouteView,
      meta: { eyebrow: 'Arcs', title: 'Arcs', showCount: true },
    },
    {
      path: '/arcs/new',
      name: 'arcsNew',
      component: EmptyRouteView,
      meta: { eyebrow: 'Arcs', title: 'New arc' },
    },
    {
      path: '/arcs/:id',
      name: 'arcDetail',
      component: EmptyRouteView,
      meta: { eyebrow: 'Arcs', title: 'Arc' },
    },
    {
      path: '/arcs/:id/edit',
      name: 'arcEdit',
      component: EmptyRouteView,
      meta: { eyebrow: 'Arcs', title: 'Edit arc' },
    },
    {
      path: '/comics',
      name: 'comics',
      component: EmptyRouteView,
      meta: { eyebrow: 'Comics', title: 'Comics', showCount: true },
    },
    {
      path: '/comics/new',
      name: 'comicsNew',
      component: EmptyRouteView,
      meta: { eyebrow: 'Comics', title: 'New comic' },
    },
    {
      path: '/comics/:id',
      name: 'comicDetail',
      component: EmptyRouteView,
      meta: { eyebrow: 'Comics', title: 'Comic' },
    },
    {
      path: '/comics/:id/edit',
      name: 'comicEdit',
      component: EmptyRouteView,
      meta: { eyebrow: 'Comics', title: 'Edit comic' },
    },
    {
      path: '/series',
      name: 'series',
      component: EmptyRouteView,
      meta: { eyebrow: 'Series', title: 'Series', showCount: true },
    },
    {
      path: '/series/:id',
      name: 'seriesDetail',
      component: EmptyRouteView,
      meta: { eyebrow: 'Series', title: 'Series' },
    },
    {
      path: '/characters',
      name: 'characters',
      component: EmptyRouteView,
      meta: { eyebrow: 'Characters', title: 'Characters', showCount: true },
    },
    {
      path: '/characters/:id',
      name: 'characterDetail',
      component: EmptyRouteView,
      meta: { eyebrow: 'Characters', title: 'Character' },
    },
    {
      path: '/collections',
      name: 'collections',
      component: EmptyRouteView,
      meta: { eyebrow: 'My collections', title: 'My collections', requiresUser: true },
    },
    {
      path: '/collections/:id',
      name: 'collectionDetail',
      component: EmptyRouteView,
      meta: { eyebrow: 'My collections', title: 'Collection', requiresUser: true },
    },
    {
      path: '/metron',
      redirect: { name: 'settings', query: { tab: 'metron' } },
    },
    {
      path: '/users',
      name: 'users',
      component: EmptyRouteView,
      meta: { eyebrow: 'Users', title: 'Manage users', requiresAdmin: true },
    },
    {
      path: '/settings',
      name: 'settings',
      component: EmptyRouteView,
      meta: { eyebrow: 'Administration', title: 'App settings', requiresAdmin: true },
    },
    {
      path: '/account',
      name: 'account',
      component: EmptyRouteView,
      meta: { eyebrow: 'Account', title: 'Account settings' },
    },
    {
      path: '/progress',
      name: 'progress',
      component: EmptyRouteView,
      meta: { eyebrow: 'Progress', title: 'Progress and achievements', requiresUser: true },
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'notFound',
      component: EmptyRouteView,
      meta: { eyebrow: 'Not Found', title: 'Page not found' },
    },
  ],
})

export function setRouteAccessContext(context) {
  Object.assign(routeAccess, { loaded: true }, context)
}

export function routeAccessRedirect(to) {
  if (!routeAccess.loaded) return null
  if (to.meta.requiresMetron && (!routeAccess.canAccessMetron || routeAccess.readOnlyGuest)) {
    return { name: 'dashboard' }
  }
  if (to.meta.requiresAdmin && !routeAccess.isAdmin) return { name: 'dashboard' }
  if (to.meta.requiresUser && !routeAccess.hasUser) return { name: 'readingOrders' }
  return null
}

router.beforeEach((to) => routeAccessRedirect(to) || true)

router.afterEach((to) => {
  const title = to.meta.title ? `${to.meta.title} - ${appTitle}` : appTitle
  document.title = title
})
