import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import DashboardView from './views/DashboardView.vue'
import PlaceholderView from './views/PlaceholderView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', component: DashboardView },
    { path: '/settings', component: PlaceholderView, props: { title: 'Settings' } },
    { path: '/logs', component: PlaceholderView, props: { title: 'Logs' } },
    { path: '/about', component: PlaceholderView, props: { title: 'About' } },
  ],
})

createApp(App).use(router).mount('#app')
