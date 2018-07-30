import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';
import React from 'react';
import { withContext } from './util/AppContext.jsx';
import './../../css/version.css';

class Version extends React.Component {
  static defaultProps = {
    error: null,
    latestVersion: '',
    productName: 'controller'
  }

  static propTypes = {
    error: PropTypes.instanceOf(Error),
    isLatest: PropTypes.bool.isRequired,
    latestVersion: PropTypes.string,
    productName: PropTypes.string,
    releaseVersion: PropTypes.string.isRequired,
  }

  renderVersionCheck = () => {
    const {latestVersion, error, isLatest} = this.props;

    if (!latestVersion) {
      return (
        <div>
          Version check failed
          {error ? `: ${error}` : ''}
        </div>
      );
    }

    if (isLatest) { return "Linkerd is up to date"; }

    return (
      <div>
        A new version ({latestVersion}) is available<br />
        <Link
          to="https://versioncheck.linkerd.io/update"
          className="button primary"
          target="_blank">
          Update Now
        </Link>
      </div>
    );
  }

  render() {
    return (
      <div className="version">
        Running {this.props.productName || "controller"} {this.props.releaseVersion}<br />
        {this.renderVersionCheck()}
      </div>
    );
  }
}

export default withContext(Version);
