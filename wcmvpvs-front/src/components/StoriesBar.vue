<template>
  <section v-if="stories.length" class="stories-bar" aria-label="Stories giocatori">
    <ul class="stories-scroll" role="list">
      <li v-for="(story, index) in stories" :key="story.id" class="story-item">
        <button
          class="story-avatar"
          :class="[
            isSeen(story.id) ? 'story-avatar--seen' : 'story-avatar--unseen',
            loadingStoryId === story.id ? 'story-avatar--loading' : '',
          ]"
          type="button"
          :aria-label="`Apri story di ${story.player_name}`"
          :disabled="Boolean(loadingStoryId)"
          @click="$emit('open', index)"
        >
          <img :src="story.thumbnail_url" :alt="story.player_name" loading="lazy" />
          <span v-if="loadingStoryId === story.id" class="story-avatar-loader" aria-hidden="true" />
        </button>
        <span v-if="showName" class="story-name">{{ compactName(story.player_name) }}</span>
      </li>
    </ul>
  </section>
</template>

<script setup>
const props = defineProps({
  stories: {
    type: Array,
    default: () => [],
  },
  seenIds: {
    type: Array,
    default: () => [],
  },
  showName: {
    type: Boolean,
    default: false,
  },
  loadingStoryId: {
    type: Number,
    default: 0,
  },
});

defineEmits(['open']);

const seenLookup = () => new Set(props.seenIds || []);

function isSeen(storyId) {
  return seenLookup().has(storyId);
}

function compactName(label) {
  const raw = String(label || '').trim();
  if (!raw) {
    return '';
  }
  const [first, second] = raw.split(/\s+/);
  if (!second) {
    return first;
  }
  return `${first} ${second.charAt(0)}.`;
}
</script>

<style scoped>
.stories-bar {
  margin-top: 0.4rem;
}

.stories-scroll {
  display: flex;
  gap: 0.7rem;
  overflow-x: auto;
  padding: 0.1rem 0.1rem 0.35rem;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
}

.stories-scroll::-webkit-scrollbar {
  display: none;
}

.story-item {
  min-width: 4.4rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.story-avatar {
  position: relative;
  border: 0;
  border-radius: 999px;
  height: 4.1rem;
  width: 4.1rem;
  padding: 2px;
  background: transparent;
}

.story-avatar:disabled {
  cursor: wait;
}

.story-avatar img {
  height: 100%;
  width: 100%;
  border-radius: 999px;
  object-fit: cover;
  border: 2px solid rgba(15, 23, 42, 0.75);
}

.story-avatar--unseen {
  background: linear-gradient(140deg, #fde047, #f97316 45%, #ec4899);
}

.story-avatar--seen {
  background: linear-gradient(140deg, #64748b, #475569);
}

.story-avatar-loader {
  position: absolute;
  inset: 0.35rem;
  border-radius: 999px;
  border: 2px solid rgba(226, 232, 240, 0.25);
  border-top-color: rgba(255, 255, 255, 0.95);
  animation: story-avatar-spin 0.85s linear infinite;
}

@keyframes story-avatar-spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.story-name {
  display: block;
  margin-top: 0.22rem;
  font-size: 0.66rem;
  color: rgba(226, 232, 240, 0.95);
  max-width: 4.3rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
