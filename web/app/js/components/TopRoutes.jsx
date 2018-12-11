import { UrlQueryParamTypes, addUrlProps } from 'react-url-query';
import Button from '@material-ui/core/Button';
import Card from '@material-ui/core/Card';
import CardContent from '@material-ui/core/CardContent';
import ConfigureProfilesMsg from './ConfigureProfilesMsg.jsx';
import Divider from '@material-ui/core/Divider';
import ErrorBanner from './ErrorBanner.jsx';
import FormControl from '@material-ui/core/FormControl';
import FormHelperText from '@material-ui/core/FormHelperText';
import Grid from '@material-ui/core/Grid';
import InputLabel from '@material-ui/core/InputLabel';
import MenuItem from '@material-ui/core/MenuItem';
import PropTypes from 'prop-types';
import QueryToCliCmd from './QueryToCliCmd.jsx';
import React from 'react';
import Select from '@material-ui/core/Select';
import TopRoutesModule from './TopRoutesModule.jsx';
import Typography from '@material-ui/core/Typography';
import _ from 'lodash';
import { groupResourcesByNs } from './util/MetricUtils.jsx';
import { withContext } from './util/AppContext.jsx';
import { withStyles } from '@material-ui/core/styles';

const topRoutesQueryProps = {
  resource_name: PropTypes.string,
  resource_type: PropTypes.string,
  namespace: PropTypes.string,
};
const topRoutesQueryPropType = PropTypes.shape(topRoutesQueryProps);

const urlPropsQueryConfig = _.mapValues(topRoutesQueryProps, () => {
  return { type: UrlQueryParamTypes.string };
});

const styles = theme => ({
  root: {
    marginTop: 3 * theme.spacing.unit,
    marginBottom:theme.spacing.unit,
  },
  formControl: {
    minWidth: 200,
  },
});
class TopRoutes extends React.Component {
  static propTypes = {
    api: PropTypes.shape({
      PrefixedLink: PropTypes.func.isRequired,
    }).isRequired,
    classes: PropTypes.shape({}).isRequired,
    query: topRoutesQueryPropType,
  }
  static defaultProps = {
    query: {
      resource_name: '',
      resource_type: '',
      namespace: '',
    },
  }

  constructor(props) {
    super(props);
    this.api = this.props.api;

    let query = _.merge({}, props.query, _.pick(this.props, _.keys(topRoutesQueryProps)));

    this.state = {
      query: query,
      error: null,
      services: [],
      namespaces: ["default"],
      resourcesByNs: {},
      pollingInterval: 5000,
      pendingRequests: false,
      requestInProgress: false
    };
  }

  componentDidMount() {
    this._isMounted = true; // https://reactjs.org/blog/2015/12/16/ismounted-antipattern.html
    this.startServerPolling();
  }

  componentWillUnmount() {
    this._isMounted = false;
    this.stopServerPolling();
  }

  loadFromServer = () => {
    if (this.state.pendingRequests) {
      return; // don't make more requests if the ones we sent haven't completed
    }
    this.setState({ pendingRequests: true });

    let allMetricsUrl = this.api.urlsForResource("all");
    this.api.setCurrentRequests([
      this.api.fetchServices(),
      this.api.fetchMetrics(allMetricsUrl)
    ]);

    this.serverPromise = Promise.all(this.api.getCurrentPromises())
      .then(([svcList, allMetrics]) => {
        let services =  _.get(svcList, 'services', []);
        let namespaces = _.uniq(_.map(services, 'namespace'));
        let { resourcesByNs } = groupResourcesByNs(allMetrics);

        this.setState({
          services,
          namespaces,
          resourcesByNs,
          pendingRequests: false,
          error: null
        });
      })
      .catch(this.handleApiError);
  }

  handleApiError = e => {
    if (e.isCanceled) {
      return;
    }

    this.setState({
      pendingRequests: false,
      error: e
    });
  }

  startServerPolling = () => {
    this.loadFromServer();
    this.timerId = window.setInterval(this.loadFromServer, this.state.pollingInterval);
  }

  stopServerPolling = () => {
    window.clearInterval(this.timerId);
    this.api.cancelCurrentRequests();
  }

  handleBtnClick = inProgress => () => {
    this.setState({
      requestInProgress: inProgress
    });
  }

  // Each time state.query is updated, this method calls the equivalent
  // onChange method to reflect the update in url query params. These onChange
  // methods are automatically added to props by react-url-query.
  handleUrlUpdate = (name, formVal) => {
    this.props[`onChange${_.upperFirst(name)}`](formVal);
  }

  handleNamespaceSelect = e => {
    let query = this.state.query;
    let formVal = _.get(e, 'target.value');
    query.namespace = formVal;
    this.handleUrlUpdate("namespace", formVal);
    this.setState({ query });
  };

  handleResourceSelect = e => {
    let query = this.state.query;
    let resource = _.get(e, 'target.value');
    let [resource_type, resource_name] = resource.split("/");
    query.resource_name = resource_name;
    query.resource_type = resource_type;
    this.handleUrlUpdate("resource_name", resource_name);
    this.handleUrlUpdate("resource_type", resource_type);
    this.setState({ query });
  }

  renderRoutesQueryForm = () => {
    const { classes } = this.props;

    return (
      <CardContent>
        <Grid container direction="column" spacing={16}>
          <Grid item container spacing={32} alignItems="center" justify="flex-start">
            <Grid item>
              { this.renderNamespaceDropdown("Namespace", "namespace", "Namespace to query") }
            </Grid>

            <Grid item>
              { this.renderServiceDropdown() }
            </Grid>

            <Grid item>
              <Button
                color="primary"
                variant="outlined"
                disabled={this.state.requestInProgress}
                onClick={this.handleBtnClick(true)}>
              Start
              </Button>
            </Grid>

            <Grid item>
              <Button
                color="default"
                variant="outlined"
                disabled={!this.state.requestInProgress}
                onClick={this.handleBtnClick(false)}>
              Stop
              </Button>
            </Grid>
          </Grid>
        </Grid>
        <Divider light className={classes.root} />
        <Typography variant="caption">You can also create a new profile <ConfigureProfilesMsg showAsIcon={true} /></Typography>
      </CardContent>
    );
  }

  renderNamespaceDropdown = (title, key, helperText) => {
    const { classes } = this.props;

    return (
      <FormControl className={classes.formControl}>
        <InputLabel htmlFor={`${key}-dropdown`}>{title}</InputLabel>
        <Select
          value={this.state.query[key]}
          onChange={this.handleNamespaceSelect}
          inputProps={{
            name: key,
            id: `${key}-dropdown`,
          }}
          name={key}>
          {
            _.map(_.sortBy(this.state.namespaces), ns =>
              <MenuItem key={`namespace-${ns}`} value={ns}>{ns}</MenuItem>)
          }
        </Select>
        <FormHelperText>{helperText}</FormHelperText>
      </FormControl>
    );
  }

  renderServiceDropdown = () => {
    const { classes } = this.props;
    let { query, services, resourcesByNs } = this.state;

    let key = "resource_name";
    let servicesWithPrefix = _.chain(services)
      .filter(['namespace', query.namespace])
      .map(svc => `service/${svc.name}`).value();
    let otherResources = resourcesByNs[query.namespace] || [];

    let dropdownOptions = _.sortBy(_.concat(servicesWithPrefix, otherResources));
    let dropdownVal = _.isEmpty(query.resource_name) || _.isEmpty(query.resource_type) ? "" :
      query.resource_type + "/" + query.resource_name;

    if (_.isEmpty(dropdownOptions) && !_.isEmpty(dropdownVal)) {
      dropdownOptions = [dropdownVal]; // populate from url if autocomplete hasn't loaded
    }

    return (
      <FormControl className={classes.formControl}>
        <InputLabel htmlFor={`${key}-dropdown`}>Resource</InputLabel>
        <Select
          value={dropdownVal}
          onChange={this.handleResourceSelect}
          disabled={_.isEmpty(query.namespace)}
          inputProps={{
            name: key,
            id: `${key}-dropdown`,
          }}
          name={key}>
          {
            _.map(dropdownOptions, resource => <MenuItem key={resource} value={resource}>{resource}</MenuItem>)
          }
        </Select>
        <FormHelperText>Resource to query</FormHelperText>
      </FormControl>
    );
  }

  render() {
    let query = this.state.query;
    let emptyQuery = _.isEmpty(query.resource_name) || _.isEmpty(query.resource_type);

    return (
      <div>
        {
          !this.state.error ? null :
          <ErrorBanner message={this.state.error} onHideMessage={() => this.setState({ error: null })} />
        }
        <Card>
          { this.renderRoutesQueryForm() }
          {  emptyQuery ? null :
          <QueryToCliCmd cmdName="routes" query={query} resource={query.resource_type + "/" + query.resource_name} /> }
          { !this.state.requestInProgress || !this._isMounted ? null : <TopRoutesModule query={this.state.query} /> }
        </Card>
      </div>
    );
  }
}

export default addUrlProps({ urlPropsQueryConfig })(withContext(withStyles(styles, { withTheme: true })(TopRoutes)));
