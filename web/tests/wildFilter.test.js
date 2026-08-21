import test from 'node:test'
import assert from 'node:assert/strict'

import { WILD_LAYERS, matchesWildPet } from '../src/pages/map/wildFilter.js'

const select = (...keys) => WILD_LAYERS.filter((layer) => keys.includes(layer.k))
const pet = (...kinds) => ({ kinds })

test('AND intersects weight and voice groups', () => {
  const and = select('weight-big', 'weight-small', 'voice-high', 'voice-low')
  assert.equal(matchesWildPet(pet('weight-big', 'voice-low'), [], [], and), true)
  assert.equal(matchesWildPet(pet('weight-small', 'voice-high'), [], [], and), true)
  assert.equal(matchesWildPet(pet('weight-big'), [], [], and), false)
  assert.equal(matchesWildPet(pet('voice-low'), [], [], and), false)
})

test('MAX conditions can participate in AND groups', () => {
  const and = select('weight-max', 'weight-min', 'voice-max', 'voice-min')
  assert.equal(matchesWildPet(pet('weight-max'), [], [], and), false)
  assert.equal(matchesWildPet(pet('voice-min'), [], [], and), false)
  assert.equal(matchesWildPet(pet('weight-max', 'voice-min'), [], [], and), true)
  assert.equal(matchesWildPet(pet('weight-min', 'voice-max'), [], [], and), true)
})

test('mutation and pollution are standalone additions', () => {
  const standalone = select('mutation', 'pollution')
  const and = select('weight-big', 'voice-high')
  for (const kind of ['shiny', 'colorful', 'pollution']) {
    assert.equal(matchesWildPet(pet(kind), standalone, [], and), true, kind)
  }
  assert.equal(matchesWildPet(pet('weight-big'), standalone, [], and), false)
})

test('OR and AND condition sets contribute simultaneously', () => {
  const or = select('weight-max', 'voice-min')
  const and = select('weight-big', 'weight-small', 'voice-high', 'voice-low')
  assert.equal(matchesWildPet(pet('weight-max'), [], or, and), true)
  assert.equal(matchesWildPet(pet('voice-min'), [], or, and), true)
  assert.equal(matchesWildPet(pet('weight-small', 'voice-high'), [], or, and), true)
  assert.equal(matchesWildPet(pet('weight-big'), [], or, and), false)
  assert.equal(matchesWildPet(pet('pollution'), [], or, and), false)
  assert.equal(matchesWildPet(pet('weight-big'), [], [], []), false)
})
