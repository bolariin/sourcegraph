import AlertCircleIcon from 'mdi-react/AlertCircleIcon'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import React from 'react'
import { Route, RouteComponentProps, Switch } from 'react-router'
import { isErrorLike } from '../../../../../shared/src/util/errors'
import { ErrorBoundary } from '../../../components/ErrorBoundary'
import { HeroPage } from '../../../components/HeroPage'
import { ThreadDiscussionPage } from '../../threadsOLD/detail/discussion/ThreadDiscussionPage'
import { ThreadSettingsPage } from '../../threadsOLD/detail/settings/ThreadSettingsPage'
import { createThreadAreaContext, ThreadAreaContext } from '../../threadsOLD/detail/ThreadArea'
import { ThreadlikeAreaSidebar } from '../../threadsOLD/sidebar/ThreadlikeAreaSidebar'
import { ChangesetsAreaContext } from '../global/ChangesetsArea'
import { useChangesetByID } from '../util/useChangesetByID'
import { useExtraChangesetInfo } from '../util/useExtraChangesetInfo'
import { ChangesetChangesPage } from './changes/ChangesetChangesPage'
import { ChangesetOperationsList } from './changes/ChangesetOperationsList'
import { ChangesetRepositoriesList } from './changes/ChangesetRepositoriesList'
import { ChangesetTasksList } from './changes/ChangesetTasksList'
import { ChangesetAreaNavbar } from './navbar/ChangesetAreaNavbar'
import { ChangesetOverview } from './overview/ChangesetOverview'

const NotFoundPage = () => (
    <HeroPage icon={MapSearchIcon} title="404: Not Found" subtitle="Sorry, the requested page was not found." />
)

interface Props extends ChangesetsAreaContext, RouteComponentProps<{ threadID: string }> {}

export interface ChangesetAreaContext extends ThreadAreaContext {}

const LOADING: 'loading' = 'loading'

/**
 * The area for a single changeset.
 */
export const ChangesetArea: React.FunctionComponent<Props> = props => {
    const [threadOrError, setThreadOrError] = useChangesetByID(props.match.params.threadID)
    const xchangeset = useExtraChangesetInfo(threadOrError)
    if (threadOrError === LOADING || xchangeset === LOADING) {
        return null // loading
    }
    if (isErrorLike(threadOrError)) {
        return <HeroPage icon={AlertCircleIcon} title="Error" subtitle={threadOrError.message} />
    }
    if (isErrorLike(xchangeset)) {
        return <HeroPage icon={AlertCircleIcon} title="Error" subtitle={xchangeset.message} />
    }
    const context = createThreadAreaContext(props, { thread: threadOrError, onThreadUpdate: setThreadOrError })
    return (
        <div className="changeset-area flex-1 d-flex overflow-hidden">
            <div className="d-flex flex-column flex-1 overflow-auto">
                <ErrorBoundary location={props.location}>
                    <ChangesetOverview
                        {...context}
                        xchangeset={xchangeset}
                        location={props.location}
                        history={props.history}
                        className="container flex-0 pb-3"
                    />
                    <div className="w-100 border-bottom" />
                    <ChangesetAreaNavbar {...context} xchangeset={xchangeset} className="flex-0 sticky-top bg-body" />
                </ErrorBoundary>
                <ErrorBoundary location={props.location}>
                    <Switch>
                        <Route
                            path={props.match.url}
                            key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                            exact={true}
                            // tslint:disable-next-line:jsx-no-lambda
                            render={routeComponentProps => (
                                <ThreadDiscussionPage
                                    {...context}
                                    {...routeComponentProps}
                                    className="container mb-3"
                                />
                            )}
                        />
                        <Route
                            path={`${props.match.url}/tasks`}
                            exact={true}
                            // tslint:disable-next-line:jsx-no-lambda
                            render={routeComponentProps => (
                                <ChangesetTasksList
                                    {...context}
                                    {...routeComponentProps}
                                    xchangeset={xchangeset}
                                    itemClassName="pl-4"
                                />
                            )}
                        />
                        <Route
                            path={`${props.match.url}/operations`}
                            exact={true}
                            // tslint:disable-next-line:jsx-no-lambda
                            render={routeComponentProps => (
                                <ChangesetOperationsList
                                    {...context}
                                    {...routeComponentProps}
                                    xchangeset={xchangeset}
                                    className="container mt-5"
                                />
                            )}
                        />
                        <Route
                            path={`${props.match.url}/commits`}
                            exact={true}
                            // tslint:disable-next-line:jsx-no-lambda
                            render={routeComponentProps => (
                                <ChangesetRepositoriesList
                                    {...context}
                                    {...routeComponentProps}
                                    xchangeset={xchangeset}
                                    showCommits={true}
                                    className="container mt-5"
                                />
                            )}
                        />
                        <Route
                            path={`${props.match.url}/changes`}
                            exact={true}
                            // tslint:disable-next-line:jsx-no-lambda
                            render={routeComponentProps => (
                                <ChangesetChangesPage
                                    {...context}
                                    {...routeComponentProps}
                                    xchangeset={xchangeset}
                                    className="container mt-5"
                                />
                            )}
                        />
                        <Route
                            path={`${props.match.url}/settings`}
                            key="hardcoded-key" // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                            exact={true}
                            // tslint:disable-next-line:jsx-no-lambda
                            render={routeComponentProps => (
                                <ThreadSettingsPage {...context} {...routeComponentProps} className="container mt-5" />
                            )}
                        />
                        <Route key="hardcoded-key" component={NotFoundPage} />
                    </Switch>
                </ErrorBoundary>
            </div>
            <ThreadlikeAreaSidebar {...context} className="changeset-area__sidebar flex-0" history={props.history} />
        </div>
    )
}
