import { grafanaIcon } from './util/SvgWrappers.jsx';
import PropTypes from 'prop-types';
import React from 'react';

const GrafanaLink = ({PrefixedLink, name, namespace, resource}) => {

  return (
    <PrefixedLink
      to={`/dashboard/db/linkerd-${resource}?var-namespace=${namespace}&var-${resource}=${name}`}
      deployment="grafana"
      targetBlank={true}>
      &nbsp;&nbsp;
      {grafanaIcon}
    </PrefixedLink>
  );
};

GrafanaLink.propTypes = {
  name: PropTypes.string.isRequired,
  namespace: PropTypes.string.isRequired,
  PrefixedLink: PropTypes.func.isRequired,
  resource: PropTypes.string.isRequired,
};

export default GrafanaLink;
