<script setup>
const labels = {
  readingOrders: {
    eyebrow: 'Reading Orders',
    title: 'Manage reading orders',
    placeholder: 'Search orders',
  },
  comics: {
    eyebrow: 'Comics',
    title: 'Comics',
    placeholder: 'Search comics',
  },
  characters: {
    eyebrow: 'Characters',
    title: 'Characters',
    placeholder: 'Search characters',
  },
  metron: {
    eyebrow: 'Metron',
    title: 'Import from Metron',
  },
}

defineProps({
  activeView: {
    type: String,
    required: true,
  },
  search: {
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

defineEmits(['update:search'])
</script>

<template>
  <header class="toolbar sticky-toolbar">
    <div>
      <p class="eyebrow">{{ labels[activeView].eyebrow }}</p>
      <h2>{{ labels[activeView].title }}</h2>
      <p v-if="activeView !== 'metron'" class="toolbar-summary">
        Showing {{ resultCount }} of {{ totalCount }}
      </p>
    </div>
    <div class="toolbar-actions">
      <div v-if="activeView !== 'metron'" class="search-field">
        <input
          :value="search"
          type="search"
          :placeholder="labels[activeView].placeholder"
          @input="$emit('update:search', $event.target.value)"
        />
        <button v-if="search" class="ghost-button" type="button" @click="$emit('update:search', '')">
          Clear
        </button>
      </div>
    </div>
  </header>
</template>
