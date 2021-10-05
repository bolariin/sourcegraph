import { useApolloClient } from '@apollo/client'
import classNames from 'classnames'
import React, { FunctionComponent, useCallback, useEffect, useMemo } from 'react'
import { RouteComponentProps } from 'react-router'
import { Subject } from 'rxjs'

import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { Container, PageHeader } from '@sourcegraph/wildcard'

import {
    FilteredConnection,
    FilteredConnectionFilter,
    FilteredConnectionQueryArguments,
} from '../../../components/FilteredConnection'
import { PageTitle } from '../../../components/PageTitle'
import { LsifIndexFields, LSIFIndexState } from '../../../graphql-operations'

import { enqueueIndexJob as defaultEnqueueIndexJob } from './backend'
import styles from './CodeIntelIndexesPage.module.scss'
import { CodeIntelIndexNode, CodeIntelIndexNodeProps } from './CodeIntelIndexNode'
import { EnqueueForm } from './EnqueueForm'
import {
    queryLsifIndexListByRepository as defaultQueryLsifIndexListByRepository,
    queryLsifIndexList as defaultQueryLsifIndexList,
} from './useLsifIndexList'

export interface CodeIntelIndexesPageProps extends RouteComponentProps<{}>, TelemetryProps {
    repo?: { id: string }
    enqueueIndexJob?: typeof defaultEnqueueIndexJob
    queryLsifIndexListByRepository?: typeof defaultQueryLsifIndexListByRepository
    queryLsifIndexList?: typeof defaultQueryLsifIndexList
    now?: () => Date
}

const filters: FilteredConnectionFilter[] = [
    {
        id: 'filters',
        label: 'Index state',
        type: 'select',
        values: [
            {
                label: 'All',
                value: 'all',
                tooltip: 'Show all indexes',
                args: {},
            },
            {
                label: 'Completed',
                value: 'completed',
                tooltip: 'Show completed indexes only',
                args: { state: LSIFIndexState.COMPLETED },
            },
            {
                label: 'Errored',
                value: 'errored',
                tooltip: 'Show errored indexes only',
                args: { state: LSIFIndexState.ERRORED },
            },
            {
                label: 'Processing',
                value: 'processing',
                tooltip: 'Show processing indexes only',
                args: { state: LSIFIndexState.PROCESSING },
            },
            {
                label: 'Queued',
                value: 'queued',
                tooltip: 'Show queued indexes only',
                args: { state: LSIFIndexState.QUEUED },
            },
        ],
    },
]

export const CodeIntelIndexesPage: FunctionComponent<CodeIntelIndexesPageProps> = ({
    repo,
    enqueueIndexJob = defaultEnqueueIndexJob,
    queryLsifIndexListByRepository = defaultQueryLsifIndexListByRepository,
    queryLsifIndexList = defaultQueryLsifIndexList,
    now,
    telemetryService,
    ...props
}) => {
    useEffect(() => telemetryService.logViewEvent('CodeIntelIndexes'), [telemetryService])
    console.log('here here here', repo?.id)
    const apolloClient = useApolloClient()
    const queryIndexes = useCallback(
        (args: FilteredConnectionQueryArguments) => {
            if (repo?.id) {
                return queryLsifIndexListByRepository(args, repo?.id, apolloClient)
            }

            return queryLsifIndexList(args, apolloClient)
        },
        [repo?.id, queryLsifIndexListByRepository, queryLsifIndexList, apolloClient]
    )

    const querySubject = useMemo(() => new Subject<string>(), [])

    return (
        <div className="code-intel-indexes">
            <PageTitle title="Auto-indexing jobs" />
            <PageHeader
                headingElement="h2"
                path={[{ text: 'Auto-indexing jobs' }]}
                description={`Auto-indexing jobs ${repo ? 'for this repository' : 'over all repositories'}.`}
                className="mb-3"
            />

            {repo && (
                <Container className="mb-2">
                    <EnqueueForm repoId={repo.id} querySubject={querySubject} enqueueIndexJob={enqueueIndexJob} />
                </Container>
            )}

            <Container>
                <div className="list-group position-relative">
                    <FilteredConnection<LsifIndexFields, Omit<CodeIntelIndexNodeProps, 'node'>>
                        listComponent="div"
                        listClassName={classNames(styles.grid, 'mb-3')}
                        noun="index"
                        pluralNoun="indexes"
                        querySubject={querySubject}
                        nodeComponent={CodeIntelIndexNode}
                        nodeComponentProps={{ now }}
                        queryConnection={queryIndexes}
                        history={props.history}
                        location={props.location}
                        cursorPaging={true}
                        filters={filters}
                    />
                </div>
            </Container>
        </div>
    )
}
