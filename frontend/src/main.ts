import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import OverviewView from './views/OverviewView.vue'
import ClientView from './views/ClientView.vue'
import PeersView from './views/PeersView.vue'
import NetworksView from './views/NetworksView.vue'
import ProfilesView from './views/ProfilesView.vue'
import DiagnosticsView from './views/DiagnosticsView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', component: OverviewView },
    { path: '/client', component: ClientView },
    { path: '/peers', component: PeersView },
    { path: '/networks', component: NetworksView },
    { path: '/profiles', component: ProfilesView },
    { path: '/diagnostics', component: DiagnosticsView },
  ],
})

createApp(App).use(router).mount('#app')
