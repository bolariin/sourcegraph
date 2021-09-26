import { subDays, startOfDay } from 'date-fns'
import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import React, { useEffect, useMemo } from 'react'

import { LoadingSpinner } from '@sourcegraph/react-loading-spinner'
import { Scalars } from '@sourcegraph/shared/src/graphql-operations'
import { useQuery } from '@sourcegraph/shared/src/graphql/apollo'
import { PageHeader } from '@sourcegraph/wildcard'

import { BatchChangesIcon } from '../../../batches/icons'
import { HeroPage } from '../../../components/HeroPage'
import { PageTitle } from '../../../components/PageTitle'
import {
    BatchChangeByNamespaceResult,
    BatchChangeByNamespaceVariables,
    BatchChangeFields,
} from '../../../graphql-operations'
import { Description } from '../Description'

import {
    queryChangesets as _queryChangesets,
    queryExternalChangesetWithFileDiffs as _queryExternalChangesetWithFileDiffs,
    queryChangesetCountsOverTime as _queryChangesetCountsOverTime,
    deleteBatchChange as _deleteBatchChange,
    queryAllChangesetIDs as _queryAllChangesetIDs,
    BATCH_CHANGE_BY_NAMESPACE,
} from './backend'
import { BatchChangeDetailsActionSection } from './BatchChangeDetailsActionSection'
import { BatchChangeDetailsProps, BatchChangeDetailsTabs } from './BatchChangeDetailsTabs'
import { BatchChangeInfoByline } from './BatchChangeInfoByline'
import { BatchChangeStatsCard } from './BatchChangeStatsCard'
import { BulkOperationsAlerts } from './BulkOperationsAlerts'
import { ChangesetsArchivedNotice } from './ChangesetsArchivedNotice'
import { ClosedNotice } from './ClosedNotice'
import { SupersedingBatchSpecAlert } from './SupersedingBatchSpecAlert'
import { UnpublishedNotice } from './UnpublishedNotice'

export interface BatchChangeDetailsPageProps extends BatchChangeDetailsProps {
    /** The namespace ID. */
    namespaceID: Scalars['ID']
    /** The batch change name. */
    batchChangeName: BatchChangeFields['name']
    /** For testing only. */
    deleteBatchChange?: typeof _deleteBatchChange
}

/**
 * The area for a single batch change.
 */
export const BatchChangeDetailsPage: React.FunctionComponent<BatchChangeDetailsPageProps> = props => {
    const { namespaceID, batchChangeName, history, location, telemetryService, deleteBatchChange } = props

    useEffect(() => {
        telemetryService.logViewEvent('BatchChangeDetailsPage')
    }, [telemetryService])

    // Query bulk operations created after this time.
    const createdAfter = useMemo(() => subDays(startOfDay(new Date()), 3).toISOString(), [])

    const { data, error, loading, refetch } = useQuery<BatchChangeByNamespaceResult, BatchChangeByNamespaceVariables>(
        BATCH_CHANGE_BY_NAMESPACE,
        {
            variables: { namespaceID, batchChange: batchChangeName, createdAfter },
            // Cache this data but always re-request it in the background when we revisit
            // this page to pick up newer changes.
            fetchPolicy: 'cache-and-network',
            // For subsequent requests while this page is open, make additional network
            // requests; this is necessary for `refetch` to actually use the network. (see
            // https://github.com/apollographql/apollo-client/issues/5515)
            nextFetchPolicy: 'network-only',
            // TODO: Why do we need to poll this every 5 seconds??
            // pollInterval: 5000,
        }
    )

    if (loading && !data) {
        return (
            <div className="text-center">
                <LoadingSpinner className="icon-inline mx-auto my-4" />
            </div>
        )
    }
    if (error || !data || !data.batchChange) {
        return <HeroPage icon={AlertCircleIcon} title="Batch change not found" />
    }

    const { batchChange } = data

    return (
        <>
            <PageTitle title={batchChange.name} />
            <PageHeader
                path={[
                    {
                        icon: BatchChangesIcon,
                        to: '/batch-changes',
                    },
                    { to: `${batchChange.namespace.url}/batch-changes`, text: batchChange.namespace.namespaceName },
                    { text: batchChange.name },
                ]}
                byline={
                    <BatchChangeInfoByline
                        createdAt={batchChange.createdAt}
                        initialApplier={batchChange.initialApplier}
                        lastAppliedAt={batchChange.lastAppliedAt}
                        lastApplier={batchChange.lastApplier}
                    />
                }
                actions={
                    <BatchChangeDetailsActionSection
                        batchChangeID={batchChange.id}
                        batchChangeClosed={!!batchChange.closedAt}
                        deleteBatchChange={deleteBatchChange}
                        batchChangeNamespaceURL={batchChange.namespace.url}
                        history={history}
                    />
                }
                className="test-batch-change-details-page mb-3"
            />
            <BulkOperationsAlerts location={location} bulkOperations={batchChange.activeBulkOperations} />
            <SupersedingBatchSpecAlert spec={batchChange.currentSpec.supersedingBatchSpec} />
            <ClosedNotice closedAt={batchChange.closedAt} className="mb-3" />
            <UnpublishedNotice
                unpublished={batchChange.changesetsStats.unpublished}
                total={batchChange.changesetsStats.total}
                className="mb-3"
            />
            <ChangesetsArchivedNotice history={history} location={location} />
            <BatchChangeStatsCard
                closedAt={batchChange.closedAt}
                stats={batchChange.changesetsStats}
                diff={batchChange.diffStat}
                className="mb-3"
            />
            <Description description={batchChange.description} />
            <button className="btn btn-outline-secondary btn-sm" onClick={() => refetch()} type="button">
                Refetch
            </button>
            <BatchChangeDetailsTabs batchChange={batchChange} refetchBatchChange={refetch} {...props} />
        </>
    )
}
