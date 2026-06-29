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
})

defineEmits(['change-view', 'refresh'])
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

    <button class="refresh-button" :disabled="loading" @click="$emit('refresh')">
      Refresh
    </button>
  </aside>
</template>
