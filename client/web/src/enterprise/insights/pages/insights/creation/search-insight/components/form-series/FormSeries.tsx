import classnames from 'classnames'
import React from 'react'

import { Button } from '@sourcegraph/wildcard/src'

import { EditableDataSeries } from '../../types'
import { FormSeriesInput } from '../form-series-input/FormSeriesInput'

import { SeriesCard } from './components/series-card/SeriesCard'
import styles from './FormSeries.module.scss'

export interface FormSeriesProps {
    /**
     * This prop represents the case whenever the edit insight UI page
     * deals with backend insight. We need to disable our search insight
     * query field since our backend insight can't update BE data according
     * to the latest insight configuration.
     */
    isBackendInsightEdit: boolean

    /**
     * Show all validation error for all forms and fields within the series forms.
     */
    showValidationErrorsOnMount: boolean

    /**
     * Controlled value (series - chart lines) for series input component.
     */
    series?: EditableDataSeries[]

    /**
     * Live change series handler while user typing in active series form.
     * Used by consumers to get latest values from series inputs and pass
     * them tp live preview chart.
     */
    onLiveChange: (liveSeries: EditableDataSeries, isValid: boolean, index: number) => void

    /**
     * Handler that runs every time user clicked edit on particular
     * series card.
     */
    onEditSeriesRequest: (editSeriesIndex: number) => void

    /**
     * Handler that runs every time use clicked commit (done) in
     * series edit form.
     */
    onEditSeriesCommit: (seriesIndex: number, editedSeries: EditableDataSeries) => void

    /**
     * Handler that runs every time use canceled (click cancel) in
     * series edit form.
     */
    onEditSeriesCancel: (closedCardIndex: number) => void

    /**
     * Handler that runs every time use removed (click remove) in
     * series card.
     */
    onSeriesRemove: (removedSeriesIndex: number) => void
}

/**
 * Renders form series (sub-form) for series (chart lines) creation code insight form.
 */
export const FormSeries: React.FunctionComponent<FormSeriesProps> = props => {
    const {
        series = [],
        isBackendInsightEdit,
        showValidationErrorsOnMount,
        onEditSeriesRequest,
        onEditSeriesCommit,
        onEditSeriesCancel,
        onSeriesRemove,
        onLiveChange,
    } = props

    return (
        <ul data-testid="form-series" className="list-unstyled d-flex flex-column">
            {series.map((line, index) =>
                line.edit ? (
                    <FormSeriesInput
                        key={line.id}
                        isSearchQueryDisabled={isBackendInsightEdit}
                        showValidationErrorsOnMount={showValidationErrorsOnMount}
                        index={index + 1}
                        cancel={series.length > 1}
                        autofocus={series.length > 1}
                        onSubmit={seriesValues => onEditSeriesCommit(index, { ...line, ...seriesValues })}
                        onCancel={() => onEditSeriesCancel(index)}
                        className={classnames('card card-body p-3', styles.formSeriesItem)}
                        onChange={(seriesValues, valid) => onLiveChange({ ...line, ...seriesValues }, valid, index)}
                        {...line}
                    />
                ) : (
                    line && (
                        <SeriesCard
                            key={`${line.id}-card`}
                            isRemoveSeriesAvailable={!isBackendInsightEdit}
                            onEdit={() => onEditSeriesRequest(index)}
                            onRemove={() => onSeriesRemove(index)}
                            className={styles.formSeriesItem}
                            {...line}
                        />
                    )
                )
            )}

            <Button
                data-testid="add-series-button"
                type="button"
                disabled={isBackendInsightEdit}
                onClick={() => onEditSeriesRequest(series.length)}
                variant="link"
                className={classnames(styles.formSeriesItem, styles.formSeriesAddButton, 'p-3')}
            >
                + Add another data series
            </Button>
        </ul>
    )
}
