import assert from 'node:assert/strict'
import test from 'node:test'

import {
  readingOrderComicsPayload,
  readingOrderDisplayComics,
  readingOrderFormFromDetail,
} from './model.js'

test('maps section entries between API detail and editor payloads', () => {
  const form = readingOrderFormFromDetail({
    id: 7,
    name: 'Event',
    description: '',
    favorite: false,
    isPublic: true,
    entries: [
      { type: 'section', section: { title: ' Main story ', description: 'Begin here.' } },
      { type: 'comic', comic: { id: 42, title: 'Issue 1', comment: '', tags: '' } },
    ],
    comics: [{ id: 42, title: 'Issue 1' }],
  })

  assert.deepEqual(form.entries[0], {
    type: 'section',
    title: ' Main story ',
    description: 'Begin here.',
  })
  assert.deepEqual(readingOrderComicsPayload(form).entries[0], {
    type: 'section',
    title: 'Main story',
    description: 'Begin here.',
  })
})

test('drops untitled sections from the save payload', () => {
  const payload = readingOrderComicsPayload({
    entries: [
      { type: 'section', title: '   ', description: 'Unused' },
      { type: 'comic', comicId: 2, comment: '', tags: '' },
    ],
  })

  assert.deepEqual(payload.entries, [{ type: 'comic', comicId: 2, comment: '', tags: '' }])
})

test('groups nested reading-order issues separately and then resumes the parent section', () => {
  const display = readingOrderDisplayComics({
    id: 7,
    comics: [
      { id: 1, read: true, skipped: false },
      { id: 2, read: false, skipped: true },
      { id: 3, read: false, skipped: false },
    ],
    entries: [
      { type: 'section', section: { title: 'Setup', description: 'Before the event.' } },
      { type: 'comic', comic: { id: 1, read: false, skipped: false } },
      {
        type: 'readingOrder',
        readingOrder: { id: 9, name: 'Tie-ins', description: 'A child order.' },
        comment: 'Read these stories next.',
        comics: [{ id: 2, read: false, skipped: false }],
      },
      { type: 'comic', comic: { id: 3, read: false, skipped: false } },
    ],
  })

  assert.equal(display.length, 3)
  assert.equal(display[0].section.label, 'Section')
  assert.equal(display[0].section.title, 'Setup')
  assert.equal(display[1].section.label, 'Reading order')
  assert.equal(display[1].section.title, 'Tie-ins')
  assert.equal(display[1].section.description, 'Read these stories next.')
  assert.equal(display[2].section.title, 'Setup')
  assert.equal(display[0].read, true)
  assert.equal(display[1].skipped, true)
})
