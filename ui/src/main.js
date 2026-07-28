import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router/index.js'
import './styles.css'
import { registerSW } from 'virtual:pwa-register'

const updateSW = registerSW({
  onNeedRefresh() {
    if (confirm('New content available. Reload?')) {
      updateSW(true)
    }
  },
  onOfflineReady() { },
})

createApp(App).use(router).mount('#app')
