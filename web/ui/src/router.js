import { createRouter, createWebHashHistory } from 'vue-router'
import DnsView from './views/DnsView.vue'
import CertView from './views/CertView.vue'
import SettingsView from './views/SettingsView.vue'

export default createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/dns' },
    { path: '/dns', component: DnsView },
    { path: '/certs', component: CertView },
    { path: '/settings', component: SettingsView }
  ]
})
