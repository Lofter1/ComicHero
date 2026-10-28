import { ref } from 'vue'
import { getAdminAnnouncement, getAnnouncement, updateAnnouncement } from '@/api/client.js'

export function useAnnouncement({ error }) {
  const announcement = ref(null)
  const announcementSettings = ref(null)
  const savingAnnouncement = ref(false)

  async function loadAnnouncement() {
    try {
      announcement.value = await getAnnouncement()
    } catch {
      // Announcements are supplementary and must not prevent the app loading.
    }
  }

  async function loadAnnouncementSettings() {
    announcementSettings.value = await getAdminAnnouncement()
  }

  async function saveAnnouncement(settings) {
    savingAnnouncement.value = true
    error.value = ''
    try {
      announcementSettings.value = await updateAnnouncement(settings)
      await loadAnnouncement()
    } catch (err) {
      error.value = err.message
    } finally {
      savingAnnouncement.value = false
    }
  }

  return {
    announcement,
    announcementSettings,
    savingAnnouncement,
    loadAnnouncement,
    loadAnnouncementSettings,
    saveAnnouncement,
  }
}
