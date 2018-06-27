import _ from 'lodash';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';
import React from 'react';
import 'whatwg-fetch';

const checkFetchOk = resp => {
  if (!resp.ok) {
    throw Error(resp.statusText);
  } else {
    return resp;
  }
};

// makeCancelable from @istarkov
// https://reactjs.org/blog/2015/12/16/ismounted-antipattern.html
const makeCancelable = (promise, onSuccess) => {
  let hasCanceled_ = false;

  const wrappedPromise = new Promise((resolve, reject) => {
    return promise.then(
      result => hasCanceled_ ? reject({ isCanceled: true }) : resolve(result),
      error => hasCanceled_ ? reject({ isCanceled: true }) : reject(error)
    );
  })
    .then(checkFetchOk)
    .then(onSuccess);

  return {
    promise: wrappedPromise,
    cancel: () => {
      hasCanceled_ = true;
    },
    status: () => {
      return hasCanceled_;
    }
  };
};

const ApiHelpers = (pathPrefix, defaultMetricsWindow = '1m') => {
  let metricsWindow = defaultMetricsWindow;
  const podsPath = `/api/pods`;

  const validMetricsWindows = {
    "10s": "10 minutes",
    "1m": "1 minute",
    "10m": "10 minutes",
    "1h": "1 hour"
  };

  const apiFetch = path => {
    if (!_.isEmpty(pathPrefix)) {
      path = `${pathPrefix}${path}`;
    }

    return makeCancelable(fetch(path), r => r.json());
  };

  const fetchMetrics = path => {
    if (path.indexOf("window") === -1) {
      if (path.indexOf("?") === -1) {
        path = `${path}?window=${getMetricsWindow()}`;
      } else {
        path = `${path}&window=${getMetricsWindow()}`;
      }
    }
    return apiFetch(path);
  };

  const fetchPods = namespace => {
    if (!_.isNil(namespace)) {
      return apiFetch(podsPath + "?namespace=" + namespace);
    }
    return apiFetch(podsPath);
  };

  const getMetricsWindow = () => metricsWindow;
  const getMetricsWindowDisplayText = () => validMetricsWindows[metricsWindow];

  const setMetricsWindow = window => {
    if (!validMetricsWindows[window]) { return; }
    metricsWindow = window;
  };

  const urlsForResource = (type, namespace) => {
    if (type === "replication_controller") {
      type = "replicationcontroller";
    }

    let baseUrl = '/api/tps-reports?resource_type=' + type;
    return !namespace ? baseUrl + '&all_namespaces=true' : baseUrl + '&namespace=' + namespace;
  };

  // maintain a list of a component's requests,
  // convenient for providing a cancel() functionality
  let currentRequests = [];
  const setCurrentRequests = cancelablePromises => {
    currentRequests = cancelablePromises;
  };
  const getCurrentPromises = () => {
    return _.map(currentRequests, 'promise');
  };
  const cancelCurrentRequests = () => {
    _.each(currentRequests, promise => {
      promise.cancel();
    });
  };

  // prefix all links in the app with `pathPrefix`
  class ConduitLink extends React.Component {
    static defaultProps = {
      deployment: "",
      targetBlank: false,
    }

    static propTypes = {
      children: PropTypes.node.isRequired,
      deployment: PropTypes.string,
      targetBlank: PropTypes.bool,
      to: PropTypes.string.isRequired,
    }

    render() {
      let prefix = pathPrefix;
      if (!_.isEmpty(this.props.deployment)) {
        prefix = prefix.replace("/web:", "/"+this.props.deployment+":");
      }
      let url = `${prefix}${this.props.to}`;

      return (
        <Link
          to={url}
          {...(this.props.targetBlank ? {target:'_blank'} : {})}>
          {this.props.children}
        </Link>
      );
    }
  }

  return {
    fetch: apiFetch,
    fetchMetrics,
    fetchPods,
    getMetricsWindow,
    setMetricsWindow,
    getValidMetricsWindows: () => _.keys(validMetricsWindows),
    getMetricsWindowDisplayText,
    urlsForResource,
    ConduitLink,
    setCurrentRequests,
    getCurrentPromises,
    cancelCurrentRequests,
    // DO NOT USE makeCancelable, use fetch, this is only exposed for testing
    makeCancelable
  };
};

export default ApiHelpers;
