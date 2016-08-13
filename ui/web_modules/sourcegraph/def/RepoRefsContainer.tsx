// tslint:disable: typedef ordered-imports

import * as React from "react";
import {Container} from "sourcegraph/Container";
import * as Dispatcher from "sourcegraph/Dispatcher";
import {DefStore} from "sourcegraph/def/DefStore";
import * as DefActions from "sourcegraph/def/DefActions";
import {Heading, List, Loader} from "sourcegraph/components/index";
import "sourcegraph/blob/BlobBackend";
import * as styles from "./styles/DefInfo.css";
import * as base from "sourcegraph/components/styles/_base.css";
import * as typography from "sourcegraph/components/styles/_typography.css";
import * as classNames from "classnames";

import {Link} from "react-router";
import {refLocsPerPage} from "sourcegraph/def/index";
import "whatwg-fetch";
import * as AnalyticsConstants from "sourcegraph/util/constants/AnalyticsConstants";
import {Repository, DownPointer} from "sourcegraph/components/symbols/index";
import {urlToRepo} from "sourcegraph/repo/routes";
import {urlToBlob} from "sourcegraph/blob/routes";

interface Props {
	repo?: string;
	rev?: string;
	commitID?: string;
	def?: string;
	defObj?: any;
	defRepos?: any[];
	refLocations: any;
}

type State = any;

export class RepoRefsContainer extends Container<Props, State> {
	static contextTypes = {
		eventLogger: React.PropTypes.object.isRequired,
	};

	constructor(props: Props) {
		super(props);
		this.state = {
			currPage: 1,
			nextPageLoading: false,
		};
		this._onNextPage = this._onNextPage.bind(this);
	}

	stores() {
		return [DefStore];
	}

	reconcileState(state: State, props: Props) {
		state.repo = props.repo || null;
		state.rev = props.rev || null;
		state.def = props.def || null;
		state.defObj = props.defObj || null;
		state.defRepos = props.defRepos || [];
		state.refLocations = props.refLocations || null;
		if (state.refLocations && state.refLocations.PagesFetched >= state.currPage) {
			state.nextPageLoading = false;
		}
	}

	onStateTransition(prevState: State, nextState: State) {
		if (nextState.currPage !== prevState.currPage || nextState.repo !== prevState.repo || nextState.rev !== prevState.rev || nextState.def !== prevState.def || nextState.defObj !== prevState.defObj) {
			Dispatcher.Backends.dispatch(new DefActions.WantRefLocations({
				repo: nextState.repo, commitID: nextState.defCommitID, def: nextState.def, repos: nextState.defRepos, page: nextState.currPage,
			}));
		}
	}

	_onNextPage() {
		let nextPage = this.state.currPage + 1;
		this.setState({currPage: nextPage, nextPageLoading: true});
		if ((this.context as any).eventLogger) {
			(this.context as any).eventLogger.logEventForCategory(AnalyticsConstants.CATEGORY_DEF_INFO, AnalyticsConstants.ACTION_CLICK, "RefsPaginatorClicked", {next_page: nextPage, repo: this.props.repo, def: this.props.def});
		}
	}

	_clickedReferencedRepo() {
		(this.context as any).eventLogger.logEventForCategory(AnalyticsConstants.CATEGORY_DEF_INFO, AnalyticsConstants.ACTION_CLICK, "ReferencedInRepoClicked", {repo: this.state.repo, def: this.state.def});
	}

	_clickedReferencedBlob() {
		(this.context as any).eventLogger.logEventForCategory(AnalyticsConstants.CATEGORY_DEF_INFO, AnalyticsConstants.ACTION_CLICK, "ReferencedInBlobClicked", {repo: this.state.repo, def: this.state.def});
	}

	render(): JSX.Element | null {
		let refLocs = this.state.refLocations;
		let fileCount = refLocs && refLocs.RepoRefs ?
			refLocs.RepoRefs.reduce((total, refs) => total + refs.Files.length, refLocs.RepoRefs[0].Files.length) : 0;

		return (
			<div>
				<Heading level="7" className={classNames(styles.cool_mid_gray, base.mb4)}>
					{refLocs && refLocs.TotalRepos &&
						`Referenced in ${refLocs.TotalRepos} repositor${refLocs.TotalRepos === 1 ? "y" : "ies"}`
					}
					{refLocs && !refLocs.TotalRepos && refLocs.RepoRefs &&
						`Referenced in ${refLocs.RepoRefs.length}+ repositor${refLocs.TotalRepos === 1 ? "y" : "ies"}`
					}
				</Heading>

				{!refLocs && <div className={typography.tc}> <Loader /></div>}
				{refLocs && !refLocs.RepoRefs && <i>No references found</i>}
				{refLocs && refLocs.RepoRefs && refLocs.RepoRefs.map((repoRefs, i) => <div key={i} className={base.mt4}>
					<Link to={urlToRepo(repoRefs.Repo)} onClick={this._clickedReferencedRepo.bind(this)} className={base.mb3}>
						<Repository width={24} className={base.mr3} /> <strong>{repoRefs.Repo}</strong>
					</Link>
					<List listStyle="node" className={base.mt2} style={{marginLeft: "6px"}}>
					{repoRefs.Files && repoRefs.Files.length > 0 && repoRefs.Files.map((file, j) =>
							<li key={j} className={classNames(base.mv3, base.pt1, styles.cool_mid_gray, styles.f7, styles.node_list_item)}>
								{file.Count} references in <Link to={urlToBlob(repoRefs.Repo, null, file.Path)} onClick={this._clickedReferencedBlob.bind(this)}>{file.Path}</Link>
							</li>)
					}
					</List>
				</div>)}
				{/* Display the paginator if we have more files repos or repos to show. */}
				{refLocs && refLocs.RepoRefs &&
					(fileCount >= refLocsPerPage || refLocs.TotalRepos > refLocs.RepoRefs.length || !refLocs.TotalRepos) &&
					!refLocs.StreamTerminated &&
					<a onClick={this._onNextPage} className={classNames(styles.f7, styles.link_icon)}>
						{this.state.nextPageLoading ?
							<strong>Loading <Loader /> </strong> :
							<strong>View more references <DownPointer className={classNames(styles.icon_cool_mid_gray, base.ml1)} width={9} /></strong>
						}
					</a>
				}
			</div>
		);
	}
}
