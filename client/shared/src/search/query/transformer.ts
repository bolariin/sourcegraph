import { replaceRange } from '../../util/strings'

import { FILTERS, FilterType } from './filters'
import { scanSearchQuery } from './scanner'
import { Filter, Token } from './token'
import { filterExists, findFilters } from './validate'

export function appendContextFilter(
    query: string,
    searchContextSpec: string | undefined,
    versionContext: string | undefined
): string {
    return !filterExists(query, FilterType.context) && searchContextSpec && !versionContext
        ? `context:${searchContextSpec} ${query}`
        : query
}

export function omitFilter(query: string, filter: Filter): string {
    return replaceRange(query, filter.range).trimStart()
}

const succeedScan = (query: string): Token[] => {
    const result = scanSearchQuery(query)
    if (result.type !== 'success') {
        throw new Error('Internal error: invariant broken: succeedScan callers must be called with a valid query')
    }
    return result.term
}

/**
 * Updates the first filter with the given value if it exists.
 * Appends a single filter at the top level of the query if it does not exist.
 * This function expects a valid query; if it is invalid it throws.
 */
export const updateFilter = (query: string, field: string, value: string): string => {
    const filters = findFilters(succeedScan(query), field)
    return filters.length > 0
        ? replaceRange(query, filters[0].range, `${field}:${value}`).trim()
        : `${query} ${field}:${value}`
}

/**
 * Updates all filters with the given value if they exist.
 * Appends a single filter at the top level of the query if none exist.
 * This function expects a valid query; if it is invalid it throws.
 */
export const updateFilters = (query: string, field: string, value: string): string => {
    const filters = findFilters(succeedScan(query), field)
    let modified = false
    for (const filter of filters.reverse()) {
        query = replaceRange(query, filter.range, `${field}:${value}`)
        modified = true
    }
    if (modified) {
        return query.trim()
    }
    return `${query} ${field}:${value}`
}

/**
 * Appends the provided filter.
 */
export const appendFilter = (query: string, field: string, value: string): string => {
    const trimmedQuery = query.trim()
    const filter = `${field}:${value}`
    return trimmedQuery.length === 0 ? filter : `${query.trimEnd()} ${filter}`
}

/**
 * Removes certain filters from a given query for privacy purposes, so query can be logged in telemtry.
 */
export const sanitizeQueryForTelemetry = (query: string): string => {
    const redactedValue = '[REDACTED]'
    const filterToRedact = [
        FilterType.repo,
        FilterType.file,
        FilterType.rev,
        FilterType.repohasfile,
        FilterType.context,
        FilterType.message,
    ]

    let newQuery = query

    for (const filter of filterToRedact) {
        if (filterExists(query, filter)) {
            newQuery = updateFilters(newQuery, filter, redactedValue)
        }
        if (filterExists(query, filter, true)) {
            newQuery = updateFilters(newQuery, `-${filter}`, redactedValue)
        }
        const alias = FILTERS[filter].alias
        if (alias) {
            if (filterExists(query, alias)) {
                newQuery = updateFilters(newQuery, alias, redactedValue)
            }
            if (filterExists(query, alias, true)) {
                newQuery = updateFilters(newQuery, `-${alias}`, redactedValue)
            }
        }
    }

    return newQuery
}
