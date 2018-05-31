import _ from 'lodash';
import BaseTable from './BaseTable.jsx';
import GrafanaLink from './GrafanaLink.jsx';
import { processedMetricsPropType } from './util/MetricUtils.js';
import PropTypes from 'prop-types';
import React from 'react';
import { Tooltip } from 'antd';
import { withContext } from './util/AppContext.jsx';
import { metricToFormatter, numericSort } from './util/Utils.js';

/*
  Table to display Success Rate, Requests and Latency in tabs.
  Expects rollup and timeseries data.
*/

const withTooltip = (d, metricName) => {
  return (
    <Tooltip
      title={metricToFormatter["UNTRUNCATED"](d)}
      overlayStyle={{ fontSize: "12px" }}>
      <span>{metricToFormatter[metricName](d)}</span>
    </Tooltip>
  );
};

const formatTitle = (title, tooltipText) => {
  if (!tooltipText) {
    return title;
  } else {
    return (
      <Tooltip
        title={tooltipText}
        overlayStyle={{ fontSize: "12px" }}>
        {title}
      </Tooltip>
    );
  }

};
const columnDefinitions = (resource, namespaces, onFilterClick, showNamespaceColumn, ConduitLink) => {
  let nsColumn = [
    {
      title: formatTitle("Namespace"),
      key: "namespace",
      dataIndex: "namespace",
      filters: namespaces,
      onFilterDropdownVisibleChange: onFilterClick,
      onFilter: (value, row) => row.namespace.indexOf(value) === 0,
      sorter: (a, b) => (a.namespace || "").localeCompare(b.namespace),
      render: ns => {
        return <ConduitLink to={"/namespaces/" + ns}>{ns}</ConduitLink>;
      }
    }
  ];

  let columns = [
    {
      title: formatTitle(resource),
      key: "name",
      defaultSortOrder: 'ascend',
      sorter: (a, b) => (a.name || "").localeCompare(b.name),
      render: row => {
        if (resource.toLowerCase() === "namespace") {
          return <ConduitLink to={"/namespaces/" + row.name}>{row.name}</ConduitLink>;
        } else if (!row.added) {
          return row.name;
        } else {
          return (
            <GrafanaLink
              name={row.name}
              namespace={row.namespace}
              resource={resource}
              ConduitLink={ConduitLink} />
          );
        }
      }
    },
    {
      title: formatTitle("SR", "Success Rate"),
      dataIndex: "successRate",
      key: "successRateRollup",
      className: "numeric",
      sorter: (a, b) => numericSort(a.successRate, b.successRate),
      render: d => metricToFormatter["SUCCESS_RATE"](d)
    },
    {
      title: formatTitle("RPS", "Request Rate"),
      dataIndex: "requestRate",
      key: "requestRateRollup",
      className: "numeric",
      sorter: (a, b) => numericSort(a.requestRate, b.requestRate),
      render: d => withTooltip(d, "REQUEST_RATE")
    },
    {
      title: formatTitle("P50", "P50 Latency"),
      dataIndex: "P50",
      key: "p50LatencyRollup",
      className: "numeric",
      sorter: (a, b) => numericSort(a.P50, b.P50),
      render: metricToFormatter["LATENCY"]
    },
    {
      title: formatTitle("P95", "P95 Latency"),
      dataIndex: "P95",
      key: "p95LatencyRollup",
      className: "numeric",
      sorter: (a, b) => numericSort(a.P95, b.P95),
      render: metricToFormatter["LATENCY"]
    },
    {
      title: formatTitle("P99", "P99 Latency"),
      dataIndex: "P99",
      key: "p99LatencyRollup",
      className: "numeric",
      sorter: (a, b) => numericSort(a.P99, b.P99),
      render: metricToFormatter["LATENCY"]
    },
    {
      title: formatTitle("Secured", "Percentage of TLS Traffic"),
      key: "securedTraffic",
      dataIndex: "meshedRequestPercent",
      className: "numeric",
      sorter: (a, b) => numericSort(a.meshedRequestPercent.get(), b.meshedRequestPercent.get()),
      render: d => _.isNil(d) ? "---" : d.prettyRate()
    }
  ];

  if (resource.toLowerCase() === "namespace" || !showNamespaceColumn) {
    return columns;
  } else {
    return _.concat(nsColumn, columns);
  }
};

/** @extends React.Component */
export class MetricsTableBase extends BaseTable {
  static defaultProps = {
    showNamespaceColumn: true,
  }

  static propTypes = {
    api: PropTypes.shape({
      ConduitLink: PropTypes.func.isRequired,
    }).isRequired,
    metrics: PropTypes.arrayOf(processedMetricsPropType.isRequired).isRequired,
    resource: PropTypes.string.isRequired,
    showNamespaceColumn: PropTypes.bool,
  }

  constructor(props) {
    super(props);
    this.api = this.props.api;
    this.onFilterDropdownVisibleChange = this.onFilterDropdownVisibleChange.bind(this);
    this.state = {
      preventTableUpdates: false
    };
  }

  shouldComponentUpdate() {
    // prevent the table from updating if the filter dropdown menu is open
    // this is because if the table updates, the filters will reset which
    // makes it impossible to select a filter
    return !this.state.preventTableUpdates;
  }

  onFilterDropdownVisibleChange(dropdownVisible) {
    this.setState({ preventTableUpdates: dropdownVisible});
  }

  preprocessMetrics() {
    let tableData = _.cloneDeep(this.props.metrics);
    let namespaces = [];

    _.each(tableData, datum => {
      namespaces.push(datum.namespace);
      _.each(datum.latency, (value, quantile) => {
        datum[quantile] = value;
      });
    });

    return {
      rows: tableData,
      namespaces: _.uniq(namespaces)
    };
  }

  render() {
    let tableData = this.preprocessMetrics();
    let namespaceFilterText = _.map(tableData.namespaces, ns => {
      return { text: ns, value: ns };
    });

    let columns = _.compact(columnDefinitions(
      this.props.resource,
      namespaceFilterText,
      this.onFilterDropdownVisibleChange,
      this.props.showNamespaceColumn,
      this.api.ConduitLink
    ));

    let locale = {
      emptyText: `No ${this.props.resource}s detected.`
    };

    return (
      <BaseTable
        dataSource={tableData.rows}
        columns={columns}
        pagination={false}
        className="conduit-table"
        rowKey={r => `${r.namespace}/${r.name}`}
        locale={locale}
        size="middle" />
    );
  }
}

export default withContext(MetricsTableBase);
