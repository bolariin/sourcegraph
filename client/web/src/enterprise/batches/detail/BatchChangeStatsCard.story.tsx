import { storiesOf } from '@storybook/react'
import React from 'react'

import { EnterpriseWebStory } from '../../components/EnterpriseWebStory'

import { BatchChangeStatsCard } from './BatchChangeStatsCard'

const { add } = storiesOf('web/batches/BatchChangeStatsCard', module).addDecorator(story => (
    <div className="p-3 container">{story()}</div>
))

add('All states', () => (
    <EnterpriseWebStory>
        {props => (
            <BatchChangeStatsCard
                {...props}
                stats={{
                    __typename: 'ChangesetsStats',
                    closed: 10,
                    deleted: 10,
                    merged: 10,
                    draft: 5,
                    open: 10,
                    total: 100,
                    archived: 18,
                    unpublished: 55,
                }}
                diff={{ added: 1000, changed: 2000, deleted: 1000, __typename: 'DiffStat' }}
                closedAt={null}
            />
        )}
    </EnterpriseWebStory>
))
add('Batch change closed', () => (
    <EnterpriseWebStory>
        {props => (
            <BatchChangeStatsCard
                {...props}
                stats={{
                    __typename: 'ChangesetsStats',
                    closed: 10,
                    deleted: 10,
                    merged: 10,
                    draft: 0,
                    open: 10,
                    archived: 18,
                    total: 100,
                    unpublished: 60,
                }}
                diff={{ added: 1000, changed: 2000, deleted: 1000, __typename: 'DiffStat' }}
                closedAt={new Date().toISOString()}
            />
        )}
    </EnterpriseWebStory>
))
add('Batch change done', () => (
    <EnterpriseWebStory>
        {props => (
            <BatchChangeStatsCard
                {...props}
                stats={{
                    __typename: 'ChangesetsStats',
                    deleted: 10,
                    closed: 10,
                    merged: 80,
                    draft: 0,
                    open: 0,
                    archived: 18,
                    total: 100,
                    unpublished: 0,
                }}
                diff={{ added: 1000, changed: 2000, deleted: 1000, __typename: 'DiffStat' }}
                closedAt={null}
            />
        )}
    </EnterpriseWebStory>
))
