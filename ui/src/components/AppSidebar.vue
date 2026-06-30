<script setup>
defineProps({
  activeView: {
    type: String,
    required: true,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  themePreference: {
    type: String,
    default: 'system',
  },
})

defineEmits(['change-view', 'refresh', 'set-theme'])
</script>

<template>
  <aside class="sidebar">
    <div>
      <h1>ComicHero</h1>
    </div>

    <nav class="nav-tabs" aria-label="Primary">
      <button :class="{ active: activeView === 'readingOrders' }" @click="$emit('change-view', 'readingOrders')">
        <span>Orders</span>
      </button>
      <button :class="{ active: activeView === 'arcs' }" @click="$emit('change-view', 'arcs')">
        <span>Arcs</span>
      </button>
      <button :class="{ active: activeView === 'comics' }" @click="$emit('change-view', 'comics')">
        <span>Comics</span>
      </button>
      <button :class="{ active: activeView === 'series' }" @click="$emit('change-view', 'series')">
        <span>Series</span>
      </button>
      <button :class="{ active: activeView === 'characters' }" @click="$emit('change-view', 'characters')">
        <span>Characters</span>
      </button>
      <button :class="{ active: activeView === 'metron' }" @click="$emit('change-view', 'metron')">
        <span>Metron</span>
      </button>
    </nav>

    <div class="sidebar-actions">
      <div class="theme-selector" role="group" aria-label="Theme">
        <button
          type="button"
          :class="{ active: themePreference === 'light' }"
          :aria-pressed="themePreference === 'light'"
          @click="$emit('set-theme', 'light')"
        >
          Light
        </button>
        <button
          type="button"
          :class="{ active: themePreference === 'dark' }"
          :aria-pressed="themePreference === 'dark'"
          @click="$emit('set-theme', 'dark')"
        >
          Dark
        </button>
        <button
          type="button"
          :class="{ active: themePreference === 'system' }"
          :aria-pressed="themePreference === 'system'"
          @click="$emit('set-theme', 'system')"
        >
          System
        </button>
      </div>

      <button class="refresh-button" :disabled="loading" @click="$emit('refresh')">
        Refresh
      </button>
    </div>
  </aside>
</template>
