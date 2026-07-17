import assert from 'node:assert/strict'
import test from 'node:test'

import {
  readingOrderComicsPayload,
  readingOrderCover,
  readingOrderDisplayComics,
  readingOrderEditorPage,
  readingOrderFormFromDetail,
  reorderReadingOrderEntry,
} from './model.js'

test('uses the computed comic cover only when no explicit reading-order cover exists', () => {
  assert.equal(
    readingOrderCover({ image: '/covers/custom.jpg', displayImage: '/covers/custom.jpg' }),
    '/covers/custom.jpg',
  )
  assert.equal(
    readingOrderCover({ image: '', displayImage: '/covers/first-comic.jpg' }),
    '/covers/first-comic.jpg',
  )
  assert.equal(readingOrderCover({ image: '', displayImage: '' }), '')
})

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

test('moves reading-order entries across editor page boundaries', () => {
  const entries = Array.from({ length: 101 }, (_, index) => ({ comicId: index + 1 }))

  const movedForward = reorderReadingOrderEntry(entries, 99, 100)
  assert.equal(movedForward[99].comicId, 101)
  assert.equal(movedForward[100].comicId, 100)

  const movedBack = reorderReadingOrderEntry(movedForward, 100, 99)
  assert.deepEqual(movedBack, entries)
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

test('pages large reading orders without dropping or renumbering entries', () => {
  const entries = Array.from({ length: 251 }, (_, index) => ({ comicId: index + 1 }))

  const first = readingOrderEditorPage(entries, 0)
  assert.equal(first.pageCount, 3)
  assert.equal(first.entries.length, 100)
  assert.equal(first.entries[0].index, 0)
  assert.equal(first.entries[99].index, 99)

  const last = readingOrderEditorPage(entries, 99)
  assert.equal(last.page, 2)
  assert.equal(last.start, 200)
  assert.equal(last.end, 251)
  assert.equal(last.entries.length, 51)
  assert.equal(last.entries[0].entry.comicId, 201)
  assert.equal(last.entries[50].index, 250)
})
