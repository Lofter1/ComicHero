<script setup>
const labels = {
  readingOrders: {
    eyebrow: 'Reading Orders',
    title: 'Manage reading orders',
  },
  arcs: {
    eyebrow: 'Arcs',
    title: 'Arcs',
  },
  comics: {
    eyebrow: 'Comics',
    title: 'Comics',
  },
  series: {
    eyebrow: 'Series',
    title: 'Series',
  },
  characters: {
    eyebrow: 'Characters',
    title: 'Characters',
  },
  metron: {
    eyebrow: 'Metron',
    title: 'Import from Metron',
  },
  progress: {
    eyebrow: 'Progress',
    title: 'Progress and achievements',
  },
  users: {
    eyebrow: 'Users',
    title: 'Manage users',
  },
  account: {
    eyebrow: 'Account',
    title: 'Account settings',
  },
}

defineProps({
  activeView: {
    type: String,
    required: true,
  },
  resultCount: {
    type: Number,
    default: 0,
  },
  totalCount: {
    type: Number,
    default: 0,
  },
})

</script>

<template>
  <header class="toolbar sticky-toolbar">
    <div>
      <p class="eyebrow">{{ labels[activeView].eyebrow }}</p>
      <h2>{{ labels[activeView].title }}</h2>
      <p v-if="!['metron', 'users', 'account', 'progress'].includes(activeView)" class="toolbar-summary">
        Showing {{ resultCount }} of {{ totalCount }}
      </p>
    </div>
  </header>
</template>
