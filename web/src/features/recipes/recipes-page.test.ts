import { describe, expect, it } from 'vitest'
import type { RecipeSummary } from '../../api/client'
import { filterRecipesByName } from './recipes-page'

const recipes = [
  recipe(1, 'Beef Pot Roast'),
  recipe(2, 'Chicken Noodle Soup'),
  recipe(3, 'Crock Pot Italian Beef Sandwiches'),
]

describe('recipe name search', () => {
  it('matches partial names without regard to case or surrounding whitespace', () => {
    expect(filterRecipesByName(recipes, '  POT ')).toEqual([recipes[0], recipes[2]])
  })

  it('returns the alphabetical source list for an empty query', () => {
    expect(filterRecipesByName(recipes, ' ')).toBe(recipes)
  })

  it('returns an empty list when no recipe name matches', () => {
    expect(filterRecipesByName(recipes, 'salmon')).toEqual([])
  })
})

function recipe(id: number, name: string): RecipeSummary {
  return {
    id,
    key: name.toLocaleLowerCase().replaceAll(' ', '-'),
    name,
    imageUrl: null,
    yield: null,
    handsOn: { minimumMinutes: null, maximumMinutes: null },
    unattended: { minimumMinutes: null, maximumMinutes: null },
  }
}
