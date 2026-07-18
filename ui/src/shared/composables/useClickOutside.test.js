import assert from 'node:assert/strict'
import test from 'node:test'

import { isClickOutside } from './useClickOutside.js'

test('treats clicks in any popup element as inside', () => {
  const trigger = { contains: (target) => target === triggerChild }
  const triggerChild = {}
  const panel = { contains: () => false }

  assert.equal(
    isClickOutside([trigger, panel], {
      target: triggerChild,
      composedPath: () => [triggerChild, trigger],
    }),
    false,
  )
})

test('treats clicks beyond all popup elements as outside', () => {
  const trigger = { contains: () => false }
  const panel = { contains: () => false }

  assert.equal(
    isClickOutside([trigger, panel], {
      target: {},
      composedPath: () => [],
    }),
    true,
  )
})
