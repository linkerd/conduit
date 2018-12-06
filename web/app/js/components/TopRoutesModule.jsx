import ConfigureProfilesMsg from './ConfigureProfilesMsg.jsx';
import ErrorBanner from './ErrorBanner.jsx';
import PropTypes from 'prop-types';
import React from 'react';
import Spinner from './util/Spinner.jsx';
import TopRoutesTable from './TopRoutesTable.jsx';
import _ from 'lodash';
import { apiErrorPropType } from './util/ApiHelpers.jsx';
import { processTopRoutesResults } from './util/MetricUtils.jsx';
import withREST from './util/withREST.jsx';

class TopRoutesBase extends React.Component {
  static defaultProps = {
    error: null
  }

  static propTypes = {
    data: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    error:  apiErrorPropType,
    loading: PropTypes.bool.isRequired,
    query: PropTypes.shape({}).isRequired,
  }

  banner = () => {
    const {error} = this.props;
    if (!error) {
      return;
    }
    return <ErrorBanner message={error} />;
  }

  loading = () => {
    const {loading} = this.props;
    if (!loading) {
      return;
    }

    return <Spinner />;
  }

  render() {
    const {data, loading} = this.props;
    let metrics = processTopRoutesResults(_.get(data, '[0].routes.rows', []));
    let allRoutesUnknown = _.every(metrics, m => m.route === "UNKNOWN");
    let showCallToAction = !loading &&  (_.isEmpty(metrics) || allRoutesUnknown);

    return (
      <React.Fragment>
        {this.loading()}
        {this.banner()}
        <TopRoutesTable rows={metrics} />
        { showCallToAction ? <ConfigureProfilesMsg /> : null}
      </React.Fragment>
    );
  }
}

export default withREST(
  TopRoutesBase,
  ({api, query}) => {
    let queryParams = new URLSearchParams(query).toString();
    return [api.fetchMetrics(`/api/routes?${queryParams}`)];
  }
);
