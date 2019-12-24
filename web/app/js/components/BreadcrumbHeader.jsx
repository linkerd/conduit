import { friendlyTitle, isResource, singularResource } from "./util/Utils.js";

import PropTypes from "prop-types";
import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import _chunk from 'lodash/chunk';
import _takeWhile from 'lodash/takeWhile';
import { withContext } from './util/AppContext.jsx';

const routeToCrumbTitle = {
  "controlplane": "Control Plane",
  "overview": "Overview",
  "tap": "Tap",
  "top": "Top",
  "routes": "Top Routes",
  "community": "Community"
};

class BreadcrumbHeader extends React.Component {
  constructor(props) {
    super(props);
    this.api = props.api;
  }

  processResourceDetailURL = segments => {
    if (segments.length === 4) {
      let splitSegments = _chunk(segments, 2);
      let resourceNameSegment = splitSegments[1];
      resourceNameSegment[0] = singularResource(resourceNameSegment[0]);
      return splitSegments[0].concat(resourceNameSegment.join('/'));
    } else {
      return segments;
    }
  }

  convertURLToBreadcrumbs = location => {
    if (location.length === 0) {
      return [];
    } else {
      let segments = location.split('/').slice(1);
      let finalSegments = this.processResourceDetailURL(segments);

      return finalSegments.map(segment => {
        let partialUrl = _takeWhile(segments, s => {
          return s !== segment;
        });

        if (partialUrl.length !== segments.length) {
          partialUrl.push(segment);
        }

        return {
          link: `/${partialUrl.join("/")}`,
          segment: segment
        };
      });
    }
  }

  segmentToFriendlyTitle = (segment, isResourceType) => {
    if (isResourceType) {
      return routeToCrumbTitle[segment] || friendlyTitle(segment).plural;
    } else {
      return routeToCrumbTitle[segment] || segment;
    }
  }

  renderBreadcrumbSegment(segment, numCrumbs, index) {
    let isMeshResource = isResource(segment);

    if (isMeshResource) {
      if (numCrumbs === 1 || index !== 0) {
        // If the segment is a K8s resource type, it should be pluralized if
        // the complete breadcrumb group describes a single-word list of
        // resources ("Namespaces") OR if the breadcrumb group describes a list
        // of resources within a specific namespace ("Namespace > linkerd >
        // Deployments")
        return this.segmentToFriendlyTitle(segment, true);
      }
      return friendlyTitle(segment).singular;
    }
    return this.segmentToFriendlyTitle(segment, false);
  }

  render() {
    const { pathPrefix, location } = this.props;
    let { pathname } = location;
    let breadcrumbs = this.convertURLToBreadcrumbs(pathname.replace(pathPrefix, ""));

    return breadcrumbs.map((pathSegment, index) => {
      return (
        <span key={pathSegment.segment}>
          {this.renderBreadcrumbSegment(pathSegment.segment, breadcrumbs.length, index)}
          { index < breadcrumbs.length - 1 ? " > " : null }
        </span>
      );
    });
  }
}

BreadcrumbHeader.propTypes = {
  api: PropTypes.shape({
    PrefixedLink: PropTypes.func.isRequired,
  }).isRequired,
  location: ReactRouterPropTypes.location.isRequired,
  pathPrefix: PropTypes.string.isRequired
};

export default withContext(BreadcrumbHeader);
