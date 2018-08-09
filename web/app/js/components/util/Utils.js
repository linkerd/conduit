import _ from 'lodash';
import * as d3 from 'd3';

/*
* Display grid constants
*/
export const baseWidth = 8; // design base width of 8px
export const rowGutter = 3 * baseWidth;

/*
* Number formatters
*/
const successRateFormatter = d3.format(".2%");
const latencyFormatter = d3.format(",");

export const formatLatencyMs = m => {
  if (_.isNil(m)) {
    return "---";
  } else {
    return `${formatLatencySec(m / 1000)}`;
  }
};

const niceLatency = l => latencyFormatter(Math.round(l));

export const formatLatencySec = latency => {
  let s = parseFloat(latency);
  if (_.isNil(s)) {
    return "---";
  } else if (s === parseFloat(0.0)) {
    return "0 s";
  } else if (s < 0.001) {
    return `${niceLatency(s * 1000 * 1000)} µs`;
  } else if (s < 1.0) {
    return `${niceLatency(s * 1000)} ms`;
  } else {
    return `${niceLatency(s)} s`;
  }
};

export const metricToFormatter = {
  "REQUEST_RATE": m => _.isNil(m) ? "---" : styleNum(m, " RPS", true),
  "SUCCESS_RATE": m => _.isNil(m) ? "---" : successRateFormatter(m),
  "LATENCY": formatLatencyMs,
  "UNTRUNCATED": m => styleNum(m, "", false)
};

/*
* Add commas to a number (converting it to a string in the process)
*/
export function addCommas(nStr) {
  nStr += '';
  let x = nStr.split('.');
  let x1 = x[0];
  let x2 = x.length > 1 ? '.' + x[1] : '';
  let rgx = /(\d+)(\d{3})/;
  while (rgx.test(x1)) {
    x1 = x1.replace(rgx, '$1' + ',' + '$2');
  }
  return x1 + x2;
}

/*
* Round a number to a given number of decimals
*/
export const roundNumber = (num, dec) => {
  return Math.round(num * Math.pow(10,dec)) / Math.pow(10,dec);
};

/*
* Shorten and style number
*/
export const styleNum = (number, unit = "", truncate = true) => {
  if (Number.isNaN(number)) {
    return "N/A";
  }

  if (truncate && number > 999999999) {
    number = roundNumber(number / 1000000000.0, 3);
    return addCommas(number) + "G" + unit;
  } else if (truncate && number > 999999) {
    number = roundNumber(number / 1000000.0, 3);
    return addCommas(number) + "M" + unit;
  } else if (truncate && number > 999) {
    number = roundNumber(number / 1000.0, 3);
    return addCommas(number) + "k" + unit;
  } else if (number > 999) {
    number = roundNumber(number, 0);
    return addCommas(number) + unit;
  } else {
    number = roundNumber(number, 2);
    return addCommas(number) + unit;
  }
};

/*
* Convert a string to a valid css class name
*/
export const toClassName = name => {
  if (!name) { return ""; }
  return _.lowerCase(name).replace(/[^a-zA-Z0-9]/g, "_");
};

/*
  Definition of sort, for ant table sorting
*/
export const numericSort = (a, b) => (_.isNil(a) ? -1 : a) - (_.isNil(b) ? -1 : b);

/*
  Nicely readable names for the stat resources
*/
export const friendlyTitle = resource => {
  let singular = _.startCase(resource);
  if (resource === "replicationcontroller") {
    singular = _.startCase("replication controller");
  }
  let titles = { singular: singular };
  if (resource === "authority") {
    titles.plural = "Authorities";
  } else {
    titles.plural = titles.singular + "s";
  }
  return titles;
};

/*
  produce octets given an ip address
*/
const decodeIPToOctets = ip => {
  ip = parseInt(ip, 10);
  return [
    ip >> 24 & 255,
    ip >> 16 & 255,
    ip >> 8 & 255,
    ip & 255
  ];
};

/*
  converts an address to an ipv4 formatted host:port pair
*/
export const publicAddressToString = (ipv4, port) => {
  let octets = decodeIPToOctets(ipv4);
  return octets.join(".") + ":" + port;
};
